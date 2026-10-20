package s3

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
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

type Service struct {
	client       *minio.Client
	ignoreFn     func(model.Item) bool
	bucket       string
	storageClass string
	partSize     uint64
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
		storageClass: config.storageClass,
		partSize:     partSize,
	}, nil
}

func (a Service) Enabled() bool {
	return a.client != nil
}

func (a Service) Name() string {
	return Name
}

func (a Service) WithIgnoreFn(ignoreFn func(model.Item) bool) model.Storage {
	a.ignoreFn = ignoreFn

	return a
}

func (a Service) Path(pathname string) string {
	return strings.TrimPrefix(pathname, "/")
}

func (a Service) Stat(ctx context.Context, pathname string) (model.Item, error) {
	if err := model.ValidPath(pathname); err != nil {
		return model.Item{}, err
	}

	realPathname := a.Path(pathname)

	if realPathname == "" {
		return rootItem(), nil
	}

	if !strings.HasSuffix(realPathname, "/") {
		info, err := a.client.StatObject(ctx, a.bucket, realPathname, minio.GetObjectOptions{})
		if err == nil {
			return convertToItem(info), nil
		}

		if !IsNotExist(err) {
			return model.Item{}, a.ConvertError(fmt.Errorf("stat object: %w", err))
		}
	}

	dirname := model.Dirname(realPathname)

	info, err := a.client.StatObject(ctx, a.bucket, dirname, minio.GetObjectOptions{})
	if err == nil {
		return convertToItem(info), nil
	}

	if !IsNotExist(err) {
		return model.Item{}, a.ConvertError(fmt.Errorf("stat object: %w", err))
	}

	exists, err := a.dirExists(ctx, dirname)
	if err != nil {
		return model.Item{}, err
	}

	if !exists {
		return model.Item{}, notExist()
	}

	return convertToItem(minio.ObjectInfo{Key: dirname}), nil
}

func (a Service) dirExists(ctx context.Context, realPathname string) (bool, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	objectsCh := a.client.ListObjects(ctx, a.bucket, minio.ListObjectsOptions{
		Prefix:  realPathname,
		MaxKeys: 1,
	})

	object, ok := <-objectsCh
	if !ok {
		return false, nil
	}

	if object.Err != nil {
		return false, a.ConvertError(fmt.Errorf("list objects: %w", object.Err))
	}

	return true, nil
}

func (a Service) List(ctx context.Context, pathname string) ([]model.Item, error) {
	if err := model.ValidPath(pathname); err != nil {
		return nil, err
	}

	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	prefix := a.Path(pathname)
	if prefix != "" {
		prefix = model.Dirname(prefix)
	}

	objectsCh := a.client.ListObjects(ctx, a.bucket, minio.ListObjectsOptions{
		Prefix: prefix,
	})

	var objects []minio.ObjectInfo

	for object := range objectsCh {
		if object.Err != nil {
			return nil, a.ConvertError(fmt.Errorf("list objects: %w", object.Err))
		}

		objects = append(objects, object)
	}

	if prefix != "" && len(objects) == 0 {
		return nil, notExist()
	}

	slices.SortFunc(objects, func(left, right minio.ObjectInfo) int {
		return strings.Compare(left.Key, right.Key)
	})

	items := make([]model.Item, 0, len(objects))

	for _, object := range objects {
		if object.Key == prefix {
			continue
		}

		item := convertToItem(object)
		if a.ignoreFn != nil && a.ignoreFn(item) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

func (a Service) WriteTo(ctx context.Context, pathname string, reader io.Reader, opts model.WriteOpts) error {
	if err := model.ValidPath(pathname); err != nil {
		return err
	}

	if strings.HasSuffix(pathname, "/") {
		return &fs.PathError{Op: "open", Path: pathname, Err: syscall.EISDIR}
	}

	if opts.Size == 0 {
		opts.Size = -1
	}

	if _, err := a.client.PutObject(ctx, a.bucket, a.Path(pathname), reader, opts.Size, minio.PutObjectOptions{
		PartSize:     a.partSize,
		StorageClass: a.storageClass,
	}); err != nil {
		return fmt.Errorf("put object: %w", err)
	}

	return nil
}

func (a Service) ReadFrom(ctx context.Context, pathname string) (model.ReadAtSeekCloser, error) {
	if err := model.ValidPath(pathname); err != nil {
		return nil, err
	}

	object, err := a.client.GetObject(ctx, a.bucket, a.Path(pathname), minio.GetObjectOptions{})
	if err != nil {
		return nil, a.ConvertError(fmt.Errorf("get object: %w", err))
	}

	if _, err = object.Stat(); err != nil {
		_ = object.Close()

		return nil, a.ConvertError(fmt.Errorf("get object: %w", err))
	}

	return object, nil
}

func (a Service) UpdateDate(_ context.Context, pathname string, _ time.Time) error {
	if err := model.ValidPath(pathname); err != nil {
		return err
	}

	return model.ErrUnsupported
}

func (a Service) Walk(ctx context.Context, pathname string, walkFn func(model.Item) error) error {
	if err := model.ValidPath(pathname); err != nil {
		return err
	}

	var ignoredDirs []string

	emit := func(item model.Item) error {
		if a.ignoreFn == nil {
			return walkFn(item)
		}

		for _, dir := range ignoredDirs {
			if strings.HasPrefix(item.Pathname, dir) {
				return nil
			}
		}

		if a.ignoreFn(item) {
			if item.IsDir() {
				ignoredDirs = append(ignoredDirs, strings.TrimSuffix(item.Pathname, "/")+"/")
			}

			return nil
		}

		return walkFn(item)
	}

	dirPrefix := a.Path(pathname)
	if dirPrefix != "" {
		dirPrefix = model.Dirname(dirPrefix)
	} else if err := emit(rootItem()); err != nil {
		return err
	}

	emitted := make(map[string]struct{})

	found, err := a.walkObjects(ctx, pathname, func(info minio.ObjectInfo, isExact bool) error {
		if isExact {
			return emit(convertToItem(info))
		}

		for _, dir := range ancestorDirs(info.Key, dirPrefix) {
			if _, ok := emitted[dir]; ok {
				continue
			}

			emitted[dir] = struct{}{}

			dirInfo := minio.ObjectInfo{Key: dir}
			if dir == info.Key {
				dirInfo = info
			}

			if err := emit(convertToItem(dirInfo)); err != nil {
				return err
			}
		}

		if strings.HasSuffix(info.Key, "/") {
			return nil
		}

		return emit(convertToItem(info))
	})
	if err != nil {
		return err
	}

	if !found && dirPrefix != "" {
		return notExist()
	}

	return nil
}

func (a Service) walkObjects(ctx context.Context, pathname string, objectFn func(info minio.ObjectInfo, isExact bool) error) (bool, error) {
	ctx, cancel := context.WithCancel(ctx)
	defer cancel()

	prefix := a.Path(pathname)

	var found bool

	if prefix != "" && !strings.HasSuffix(prefix, "/") {
		info, err := a.client.StatObject(ctx, a.bucket, prefix, minio.GetObjectOptions{})
		if err == nil {
			found = true

			if err = objectFn(info, true); err != nil {
				return found, err
			}
		} else if !IsNotExist(err) {
			return false, a.ConvertError(fmt.Errorf("stat object: %w", err))
		}

		prefix = model.Dirname(prefix)
	}

	objectsCh := a.client.ListObjects(ctx, a.bucket, minio.ListObjectsOptions{
		Prefix:    prefix,
		Recursive: true,
	})

	var err error

	for object := range objectsCh {
		if err != nil {
			continue
		}

		if object.Err != nil {
			err = a.ConvertError(fmt.Errorf("list objects: %w", object.Err))
			cancel()

			continue
		}

		found = true

		if err = objectFn(object, false); err != nil {
			cancel()
		}
	}

	return found, err
}

func (a Service) Mkdir(ctx context.Context, name string, _ os.FileMode) error {
	if err := model.ValidPath(name); err != nil {
		return err
	}

	trimmed := strings.Trim(a.Path(name), "/")
	if trimmed == "" {
		return nil
	}

	parts := strings.Split(trimmed, "/")

	for index := range parts {
		key := strings.Join(parts[:index+1], "/")

		_, err := a.client.StatObject(ctx, a.bucket, key, minio.GetObjectOptions{})
		if err == nil {
			if index == len(parts)-1 {
				return &fs.PathError{Op: "mkdir", Path: key, Err: syscall.EEXIST}
			}

			return model.ErrNotExist(&fs.PathError{Op: "mkdir", Path: key, Err: syscall.ENOTDIR})
		}

		if !IsNotExist(err) {
			return a.ConvertError(fmt.Errorf("stat object: %w", err))
		}
	}

	dirname := model.Dirname(trimmed)

	if _, err := a.client.StatObject(ctx, a.bucket, dirname, minio.GetObjectOptions{}); err == nil {
		return nil
	} else if !IsNotExist(err) {
		return a.ConvertError(fmt.Errorf("stat object: %w", err))
	}

	if _, err := a.client.PutObject(ctx, a.bucket, dirname, strings.NewReader(""), 0, minio.PutObjectOptions{
		StorageClass: a.storageClass,
	}); err != nil {
		return a.ConvertError(fmt.Errorf("create directory: %w", err))
	}

	return nil
}

func (a Service) Rename(ctx context.Context, oldName, newName string) error {
	if err := model.ValidPath(oldName); err != nil {
		return err
	}

	if err := model.ValidPath(newName); err != nil {
		return err
	}

	oldPrefix := strings.Trim(a.Path(oldName), "/")
	newPrefix := strings.Trim(a.Path(newName), "/")

	if oldPrefix == "" || newPrefix == "" || strings.HasPrefix(newPrefix, oldPrefix+"/") {
		return &fs.PathError{Op: "rename", Path: oldName, Err: syscall.EINVAL}
	}

	if oldPrefix == newPrefix {
		_, err := a.Stat(ctx, oldName)

		return err
	}

	found, err := a.walkObjects(ctx, oldName, func(info minio.ObjectInfo, _ bool) error {
		_, err := a.client.CopyObject(ctx, minio.CopyDestOptions{
			Bucket: a.bucket,
			Object: newPrefix + strings.TrimPrefix(info.Key, oldPrefix),
		}, minio.CopySrcOptions{
			Bucket: a.bucket,
			Object: info.Key,
		})
		if err != nil {
			return a.ConvertError(fmt.Errorf("copy object: %w", err))
		}

		if err = a.client.RemoveObject(ctx, a.bucket, info.Key, minio.RemoveObjectOptions{}); err != nil {
			return a.ConvertError(fmt.Errorf("delete object: %w", err))
		}

		return nil
	})
	if err != nil {
		return err
	}

	if !found {
		return notExist()
	}

	return nil
}

func (a Service) RemoveAll(ctx context.Context, name string) error {
	if err := model.ValidPath(name); err != nil {
		return err
	}

	if strings.Trim(a.Path(name), "/") == "" {
		return &fs.PathError{Op: "RemoveAll", Path: name, Err: syscall.EINVAL}
	}

	_, err := a.walkObjects(ctx, name, func(info minio.ObjectInfo, _ bool) error {
		if err := a.client.RemoveObject(ctx, a.bucket, info.Key, minio.RemoveObjectOptions{}); err != nil {
			return a.ConvertError(fmt.Errorf("delete object: %w", err))
		}

		return nil
	})

	return err
}

func IsNotExist(err error) bool {
	var response minio.ErrorResponse

	return errors.As(err, &response) && response.Code == minio.NoSuchKey
}

func (a Service) ConvertError(err error) error {
	if err == nil {
		return nil
	}

	if IsNotExist(err) {
		return model.ErrNotExist(err)
	}

	return err
}

func notExist() error {
	return model.ErrNotExist(fs.ErrNotExist)
}

func ancestorDirs(key, start string) []string {
	var dirs []string

	for index := 0; index < len(key); index++ {
		if key[index] != '/' || index+1 < len(start) {
			continue
		}

		dirs = append(dirs, key[:index+1])
	}

	return dirs
}
