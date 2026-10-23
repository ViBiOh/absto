package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"path"
	"slices"
	"strings"
	"syscall"
	"time"

	"github.com/ViBiOh/absto/pkg/model"
	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

const Name = "object"

var _ model.Storage = Service{}

type Config struct {
	region       string
	storageClass string
	prefix       string
}

type ConfigOption func(Config) Config

func WithRegion(region string) ConfigOption {
	return func(instance Config) Config {
		instance.region = region

		return instance
	}
}

func WithStorageClass(storageClass string) ConfigOption {
	return func(instance Config) Config {
		instance.storageClass = storageClass

		return instance
	}
}

func WithPrefix(prefix string) ConfigOption {
	return func(instance Config) Config {
		if len(prefix) != 0 {
			instance.prefix = strings.TrimPrefix(path.Clean(prefix), "/")
		}

		return instance
	}
}

type Service struct {
	client       *minio.Client
	ignoreFn     func(model.Item) bool
	bucket       string
	prefix       string
	storageClass string
	partSize     uint64
	enabled      bool
}

func New(endpoint, accessKey, secretAccess, bucket string, useSSL bool, partSize uint64, options ...ConfigOption) (Service, error) {
	if len(endpoint) == 0 {
		return Service{}, nil
	}

	var config Config
	for _, option := range options {
		config = option(config)
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretAccess, ""),
		Secure: useSSL,
		Region: config.region,
	})
	if err != nil {
		return Service{}, fmt.Errorf("minio client: %w", err)
	}

	return Service{
		client:       client,
		bucket:       bucket,
		prefix:       config.prefix,
		storageClass: config.storageClass,
		partSize:     partSize,
		enabled:      true,
	}, nil
}

func (s Service) Name() string {
	return Name
}

func (s Service) WithIgnoreFn(ignoreFn func(model.Item) bool) model.Storage {
	s.ignoreFn = ignoreFn

	return s
}

func (s Service) Stat(ctx context.Context, pathname string) (model.Item, error) {
	if !s.enabled {
		return model.Item{}, model.ErrNotEnabled
	}

	if err := model.ValidPath(pathname); err != nil {
		return model.Item{}, err
	}

	return s.stat(ctx, pathname)
}

func (s Service) stat(ctx context.Context, pathname string) (model.Item, error) {
	realPathname := s.relative(pathname)

	if s.isRoot(realPathname) {
		return rootItem(), nil
	}

	info, err := s.client.StatObject(ctx, s.bucket, realPathname, minio.GetObjectOptions{})
	if err == nil {
		return s.convertToItem(info), nil
	}

	if !IsNotExist(err) {
		return model.Item{}, s.ConvertError(err)
	}

	return s.statDir(ctx, realPathname)
}

func (s Service) statDir(ctx context.Context, name string) (model.Item, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	name = dirKey(name)

	objects := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:  name,
		MaxKeys: 1,
	})

	result, ok := <-objects
	if !ok {
		return model.Item{}, notExist()
	}

	if result.Err != nil {
		return model.Item{}, s.ConvertError(result.Err)
	}

	return s.convertToItem(minio.ObjectInfo{Key: name}), nil
}

func (s Service) List(ctx context.Context, pathname string) ([]model.Item, error) {
	if !s.enabled {
		return nil, model.ErrNotEnabled
	}

	if err := model.ValidPath(pathname); err != nil {
		return nil, err
	}

	prefix := s.relative(model.Dirname(pathname))

	items, exists, err := s.list(ctx, prefix)
	if err != nil {
		return nil, err
	}

	if !exists && !s.isRoot(prefix) {
		return nil, notExist()
	}

	return items, nil
}

func (s Service) list(ctx context.Context, prefix string) ([]model.Item, bool, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	objectsCh := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:   prefix,
		Unsorted: true,
	})

	var items []model.Item

	var exists bool

	for object := range objectsCh {
		if object.Err != nil {
			return nil, false, s.ConvertError(fmt.Errorf("list objects: %w", object.Err))
		}

		exists = true

		if object.Key == prefix {
			continue
		}

		item := s.convertToItem(object)
		if s.ignoreFn != nil && s.ignoreFn(item) {
			continue
		}

		items = append(items, item)
	}

	slices.SortFunc(items, func(left, right model.Item) int {
		return strings.Compare(left.SortKey(), right.SortKey())
	})

	return items, exists, nil
}

func (s Service) WriteTo(ctx context.Context, pathname string, reader io.Reader, opts model.WriteOpts) error {
	if !s.enabled {
		return model.ErrNotEnabled
	}

	if err := model.ValidPath(pathname); err != nil {
		return err
	}

	if strings.HasSuffix(pathname, "/") {
		return &fs.PathError{Op: "open", Path: pathname, Err: syscall.EISDIR}
	}

	if opts.Size == 0 {
		opts.Size = -1
	}

	if _, err := s.client.PutObject(ctx, s.bucket, s.relative(pathname), reader, opts.Size, minio.PutObjectOptions{
		PartSize:     s.partSize,
		StorageClass: s.storageClass,
	}); err != nil {
		return fmt.Errorf("put object: %w", err)
	}

	return nil
}

func (s Service) ReadFrom(ctx context.Context, pathname string) (model.ReadAtSeekCloser, error) {
	if !s.enabled {
		return nil, model.ErrNotEnabled
	}

	if err := model.ValidPath(pathname); err != nil {
		return nil, err
	}

	if strings.HasSuffix(pathname, "/") {
		return nil, &fs.PathError{Op: "open", Path: pathname, Err: syscall.EISDIR}
	}

	object, err := s.client.GetObject(ctx, s.bucket, s.relative(pathname), minio.GetObjectOptions{})
	if err != nil {
		return nil, s.ConvertError(fmt.Errorf("get object: %w", err))
	}

	if _, err = object.Stat(); err != nil {
		_ = object.Close()

		return nil, s.ConvertError(fmt.Errorf("get object: %w", err))
	}

	return object, nil
}

func (s Service) UpdateDate(_ context.Context, pathname string, _ time.Time) error {
	if !s.enabled {
		return model.ErrNotEnabled
	}

	if err := model.ValidPath(pathname); err != nil {
		return err
	}

	return model.ErrUnsupported
}

func (s Service) Walk(ctx context.Context, pathname string, walkFn func(model.Item) error) error {
	if !s.enabled {
		return model.ErrNotEnabled
	}

	if err := model.ValidPath(pathname); err != nil {
		return err
	}

	item, err := s.stat(ctx, pathname)
	if err != nil {
		return err
	}

	if !item.IsDir() {
		return model.ErrDirExpected
	}

	if s.ignoreFn != nil && s.ignoreFn(item) {
		return nil
	}

	if err = walkFn(item); err != nil {
		return err
	}

	return s.walkObjects(ctx, s.relative(item.Pathname), walkFn)
}

func (s Service) walkObjects(ctx context.Context, key string, walkFn func(model.Item) error) error {
	items, _, err := s.list(ctx, dirKey(key))
	if err != nil {
		return err
	}

	for _, item := range items {
		if err = walkFn(item); err != nil {
			return err
		}

		if !item.IsDir() {
			continue
		}

		if err = s.walkObjects(ctx, s.relative(item.Pathname), walkFn); err != nil {
			return err
		}
	}

	return nil
}

func (s Service) Mkdir(ctx context.Context, pathname string) error {
	if !s.enabled {
		return model.ErrNotEnabled
	}

	if err := model.ValidPath(pathname); err != nil {
		return err
	}

	if item, err := s.stat(ctx, pathname); err == nil {
		if item.IsDir() {
			return nil
		}

		return &fs.PathError{Op: "mkdir", Path: pathname, Err: syscall.EEXIST}
	} else if !model.IsNotExist(err) {
		return err
	}

	for parent := path.Dir(path.Clean("/" + pathname)); parent != "/"; parent = path.Dir(parent) {
		item, err := s.stat(ctx, parent)
		if err != nil {
			if model.IsNotExist(err) {
				continue
			}

			return err
		}

		if !item.IsDir() {
			return model.ErrNotExist(&fs.PathError{Op: "mkdir", Path: parent, Err: syscall.ENOTDIR})
		}

		break
	}

	if _, err := s.client.PutObject(ctx, s.bucket, model.Dirname(s.relative(pathname)), strings.NewReader(""), 0, minio.PutObjectOptions{StorageClass: s.storageClass}); err != nil {
		return s.ConvertError(fmt.Errorf("create directory: %w", err))
	}

	return nil
}

func (s Service) Rename(ctx context.Context, oldName, newName string) error {
	if !s.enabled {
		return model.ErrNotEnabled
	}

	if err := model.ValidPath(oldName); err != nil {
		return err
	}

	if err := model.ValidPath(newName); err != nil {
		return err
	}

	oldClean := path.Clean("/" + oldName)
	newClean := path.Clean("/" + newName)

	if oldClean == "/" || newClean == "/" || strings.HasPrefix(newClean, oldClean+"/") {
		return &fs.PathError{Op: "rename", Path: oldName, Err: syscall.EINVAL}
	}

	item, err := s.stat(ctx, oldName)
	if err != nil {
		return err
	}

	if oldClean == newClean {
		return nil
	}

	oldRelative := s.relative(oldClean)
	newRelative := s.relative(newClean)

	if !item.IsDir() {
		return s.rename(ctx, oldRelative, newRelative, oldRelative)
	}

	err = s.walkRecursive(ctx, model.Dirname(oldClean), func(object minio.ObjectInfo) error {
		return s.rename(ctx, oldRelative, newRelative, object.Key)
	})

	return s.ConvertError(err)
}

func (s Service) rename(ctx context.Context, oldName, newName, objectKey string) error {
	_, err := s.client.CopyObject(ctx, minio.CopyDestOptions{
		Bucket: s.bucket,
		Object: newName + strings.TrimPrefix(objectKey, oldName),
	}, minio.CopySrcOptions{
		Bucket: s.bucket,
		Object: objectKey,
	})
	if err != nil {
		return s.ConvertError(fmt.Errorf("copy object: %w", err))
	}

	if err = s.client.RemoveObject(ctx, s.bucket, objectKey, minio.RemoveObjectOptions{}); err != nil {
		return s.ConvertError(fmt.Errorf("delete object: %w", err))
	}

	return nil
}

func (s Service) RemoveAll(ctx context.Context, pathname string) error {
	if !s.enabled {
		return model.ErrNotEnabled
	}

	if err := model.ValidPath(pathname); err != nil {
		return err
	}

	item, err := s.stat(ctx, pathname)
	if err != nil {
		if model.IsNotExist(err) {
			return nil
		}

		return err
	}

	if !item.IsDir() {
		return s.ConvertError(s.client.RemoveObject(ctx, s.bucket, s.relative(item.Pathname), minio.RemoveObjectOptions{}))
	}

	if item.Pathname == "/" {
		return &fs.PathError{Op: "RemoveAll", Path: pathname, Err: syscall.EINVAL}
	}

	var walkErr error

	walkIter := func(yield func(minio.ObjectInfo) bool) {
		walkErr = s.walkRecursive(ctx, model.Dirname(item.Pathname), func(info minio.ObjectInfo) error {
			if !yield(info) {
				return errStopWalk
			}

			return nil
		})

		if errors.Is(walkErr, errStopWalk) {
			walkErr = nil
		}
	}

	results, err := s.client.RemoveObjectsWithIter(ctx, s.bucket, walkIter, minio.RemoveObjectsOptions{})
	if err != nil {
		return fmt.Errorf("remove iter: %w", s.ConvertError(err))
	}

	for resultErr := range results {
		err = errors.Join(err, s.ConvertError(resultErr.Err))
	}

	return errors.Join(err, walkErr)
}

func (s Service) walkRecursive(ctx context.Context, pathname string, walkFn func(minio.ObjectInfo) error) error {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	objectsCh := s.client.ListObjects(ctx, s.bucket, minio.ListObjectsOptions{
		Prefix:    s.relative(pathname),
		Recursive: true,
	})

	for object := range objectsCh {
		if object.Err != nil {
			return fmt.Errorf("list objects: %w", s.ConvertError(object.Err))
		}

		if err := walkFn(object); err != nil {
			return err
		}
	}

	return nil
}
