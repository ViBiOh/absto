package s3

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/ViBiOh/absto/pkg/model"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

// Name of the storage implementation
const Name = "s3"

var _ model.Storage = App{}

// App of package
type App struct {
	client   *minio.Client
	ignoreFn func(model.Item) bool
	bucket   string
}

// New creates new App from Config
func New(endpoint, accessKey, secretAccess, bucket string, useSSL bool) (App, error) {
	if len(endpoint) == 0 {
		return App{}, nil
	}

	client, err := minio.New(endpoint, &minio.Options{
		Creds:  credentials.NewStaticV4(accessKey, secretAccess, ""),
		Secure: useSSL,
	})
	if err != nil {
		return App{}, fmt.Errorf("unable to create minio client: %s", err)
	}

	return App{
		client: client,
		bucket: bucket,
	}, nil
}

// Enabled checks that requirements are met
func (a App) Enabled() bool {
	return a.client != nil
}

// Name of the sotrage
func (a App) Name() string {
	return Name
}

// WithIgnoreFn create a new App with given ignoreFn
func (a App) WithIgnoreFn(ignoreFn func(model.Item) bool) model.Storage {
	a.ignoreFn = ignoreFn

	return a
}

// Path return full path of pathname
func (a App) Path(pathname string) string {
	return strings.TrimPrefix(pathname, "/")
}

// Info provide metadata about given pathname
func (a App) Info(ctx context.Context, pathname string) (model.Item, error) {
	realPathname := a.Path(pathname)

	if realPathname == "" {
		return model.Item{
			Name:     "/",
			Pathname: "/",
			IsDir:    true,
		}, nil
	}

	info, err := a.client.StatObject(ctx, a.bucket, realPathname, minio.GetObjectOptions{})
	if err != nil {
		return model.Item{}, convertError(fmt.Errorf("unable to stat object: %s", err))
	}

	return convertToItem(info), nil
}

// List items in the storage
func (a App) List(ctx context.Context, pathname string) ([]model.Item, error) {
	realPathname := a.Path(pathname)
	baseRealPathname := path.Base(realPathname)

	objectsCh := a.client.ListObjects(ctx, a.bucket, minio.ListObjectsOptions{
		Prefix: realPathname,
	})

	var items []model.Item
	for object := range objectsCh {
		item := convertToItem(object)
		if item.IsDir && item.Name == baseRealPathname {
			continue
		}

		if a.ignoreFn != nil && a.ignoreFn(item) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

// WriteTo with content from reader to pathname
func (a App) WriteTo(ctx context.Context, pathname string, reader io.Reader) error {
	if _, err := a.client.PutObject(ctx, a.bucket, a.Path(pathname), reader, -1, minio.PutObjectOptions{}); err != nil {
		return fmt.Errorf("unable to put object: %s", err)
	}

	return nil
}

// ReadFrom reads content from given pathname
func (a App) ReadFrom(ctx context.Context, pathname string) (io.ReadSeekCloser, error) {
	object, err := a.client.GetObject(ctx, a.bucket, a.Path(pathname), minio.GetObjectOptions{})
	if err != nil {
		return nil, convertError(fmt.Errorf("unable to get object: %s", err))
	}

	return object, nil
}

// UpdateDate update date from given value
func (a App) UpdateDate(ctx context.Context, pathname string, date time.Time) error {
	// TODO
	return nil
}

// Walk browses item recursively
func (a App) Walk(ctx context.Context, pathname string, walkFn func(model.Item) error) error {
	objectsCh := a.client.ListObjects(ctx, a.bucket, minio.ListObjectsOptions{
		Prefix:    a.Path(pathname),
		Recursive: true,
	})

	for object := range objectsCh {
		item := convertToItem(object)
		if a.ignoreFn != nil && a.ignoreFn(item) {
			continue
		}

		if err := walkFn(item); err != nil {
			return err
		}
	}

	return nil
}

// CreateDir container in storage
func (a App) CreateDir(ctx context.Context, name string) error {
	_, err := a.client.PutObject(ctx, a.bucket, dirname(a.Path(name)), strings.NewReader(""), 0, minio.PutObjectOptions{})
	if err != nil {
		return convertError(fmt.Errorf("unable to create directory: %s", err))
	}

	return nil
}

// Rename file or directory from storage
func (a App) Rename(ctx context.Context, oldName, newName string) error {
	oldRoot := a.Path(oldName)
	newRoot := a.Path(newName)

	return a.Walk(ctx, oldRoot, func(item model.Item) error {
		pathname := a.Path(item.Pathname)

		_, err := a.client.CopyObject(ctx, minio.CopyDestOptions{
			Bucket: a.bucket,
			Object: strings.Replace(pathname, oldRoot, newRoot, -1),
		}, minio.CopySrcOptions{
			Bucket: a.bucket,
			Object: pathname,
		})
		if err != nil {
			return convertError(err)
		}

		if err = a.client.RemoveObject(ctx, a.bucket, pathname, minio.RemoveObjectOptions{}); err != nil {
			return convertError(fmt.Errorf("unable to delete object: %s", err))
		}

		return nil
	})
}

// Remove file or directory from storage
func (a App) Remove(ctx context.Context, pathname string) error {
	return a.Walk(ctx, pathname, func(item model.Item) error {
		if err := a.client.RemoveObject(ctx, a.bucket, a.Path(item.Pathname), minio.RemoveObjectOptions{}); err != nil {
			return convertError(fmt.Errorf("unable to delete object: %s", err))
		}

		return nil
	})
}
