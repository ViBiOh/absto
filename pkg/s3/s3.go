package s3

import (
	"context"
	"fmt"
	"io"
	"os"
	"path"
	"strings"
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
	realPathname := a.Path(pathname)

	if realPathname == "" {
		return model.Item{
			ID:         model.ID(pathname),
			NameValue:  "/",
			Pathname:   "/",
			IsDirValue: true,
		}, nil
	}

	info, err := a.client.StatObject(ctx, a.bucket, realPathname, minio.GetObjectOptions{})
	if err != nil {
		if strings.HasSuffix(realPathname, "/") && IsNotExist(err) && a.dirExists(ctx, realPathname) {
			return convertToItem(minio.ObjectInfo{Key: realPathname}), nil
		}

		return model.Item{}, a.ConvertError(fmt.Errorf("stat object `%s`: %w", pathname, err))
	}

	return convertToItem(info), nil
}

func (a Service) dirExists(ctx context.Context, realPathname string) bool {
	objectsCh := a.client.ListObjects(ctx, a.bucket, minio.ListObjectsOptions{
		Prefix:  realPathname,
		MaxKeys: 1,
	})

	var found uint
	for range objectsCh {
		found++
	}

	return found > 0
}

func (a Service) List(ctx context.Context, pathname string) ([]model.Item, error) {
	realPathname := a.Path(pathname)
	baseRealPathname := path.Base(realPathname)

	objectsCh := a.client.ListObjects(ctx, a.bucket, minio.ListObjectsOptions{
		Prefix: realPathname,
	})

	var items []model.Item
	for object := range objectsCh {
		item := convertToItem(object)
		if item.IsDir() && item.Name() == baseRealPathname {
			continue
		}

		if a.ignoreFn != nil && a.ignoreFn(item) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

func (a Service) WriteTo(ctx context.Context, pathname string, reader io.Reader, opts model.WriteOpts) error {
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
	object, err := a.client.GetObject(ctx, a.bucket, a.Path(pathname), minio.GetObjectOptions{})
	if err != nil {
		return nil, a.ConvertError(fmt.Errorf("get object `%s`: %w", pathname, err))
	}

	return object, nil
}

func (a Service) UpdateDate(_ context.Context, _ string, _ time.Time) error {
	return nil
}

func (a Service) Walk(ctx context.Context, pathname string, walkFn func(model.Item) error) error {
	objectsCh := a.client.ListObjects(ctx, a.bucket, minio.ListObjectsOptions{
		Prefix:    a.Path(pathname),
		Recursive: true,
	})

	var err error

	for object := range objectsCh {
		if err != nil {
			continue
		}

		item := convertToItem(object)
		if a.ignoreFn != nil && a.ignoreFn(item) {
			continue
		}

		err = walkFn(item)
	}

	return err
}

func (a Service) Mkdir(ctx context.Context, name string, _ os.FileMode) error {
	parts := strings.Split(model.Dirname(a.Path(name)), "/")

	for index := range parts {
		dirname := strings.Join(parts[:index], "/")

		if _, err := a.Stat(ctx, dirname); err != nil {
			if !model.IsNotExist(err) {
				return fmt.Errorf("info `%s`: %w", dirname, err)
			}

			if _, err = a.client.PutObject(ctx, a.bucket, model.Dirname(dirname), strings.NewReader(""), 0, minio.PutObjectOptions{
				StorageClass: a.storageClass,
			}); err != nil {
				return a.ConvertError(fmt.Errorf("create directory: %w", err))
			}
		}
	}

	return nil
}

func (a Service) Rename(ctx context.Context, oldName, newName string) error {
	oldRoot := a.Path(oldName)
	newRoot := a.Path(newName)

	return a.Walk(ctx, oldRoot, func(item model.Item) error {
		pathname := a.Path(item.Pathname)

		if item.IsDir() {
			pathname = model.Dirname(pathname)
		}

		_, err := a.client.CopyObject(ctx, minio.CopyDestOptions{
			Bucket: a.bucket,
			Object: strings.Replace(pathname, oldRoot, newRoot, 1),
		}, minio.CopySrcOptions{
			Bucket: a.bucket,
			Object: pathname,
		})
		if err != nil {
			return a.ConvertError(err)
		}

		if err = a.client.RemoveObject(ctx, a.bucket, pathname, minio.RemoveObjectOptions{}); err != nil {
			return a.ConvertError(fmt.Errorf("delete object `%s`: %w", pathname, err))
		}

		return nil
	})
}

func (a Service) RemoveAll(ctx context.Context, name string) error {
	if err := a.Walk(ctx, name, func(item model.Item) error {
		if err := a.client.RemoveObject(ctx, a.bucket, a.Path(item.Pathname), minio.RemoveObjectOptions{}); err != nil {
			return a.ConvertError(fmt.Errorf("delete object `%s`: %w", item.Pathname, err))
		}

		return nil
	}); err != nil {
		return err
	}

	return a.client.RemoveObject(ctx, a.bucket, a.Path(name), minio.RemoveObjectOptions{})
}

func IsNotExist(err error) bool {
	return strings.Contains(err.Error(), "The specified key does not exist")
}

func (a Service) ConvertError(err error) error {
	if err == nil {
		return err
	}

	if IsNotExist(err) {
		return model.ErrNotExist(err)
	}

	return err
}
