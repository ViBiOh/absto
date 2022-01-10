package s3

import (
	"context"
	"fmt"
	"io"
	"path"
	"strings"
	"time"

	"github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/logger"

	"github.com/minio/minio-go/v7"
	"github.com/minio/minio-go/v7/pkg/credentials"
)

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

// WithIgnoreFn create a new App with given ignoreFn
func (a App) WithIgnoreFn(ignoreFn func(model.Item) bool) model.Storage {
	a.ignoreFn = ignoreFn

	return a
}

// Info provide metadata about given pathname
func (a App) Info(pathname string) (model.Item, error) {
	realPathname := getPath(pathname)

	if realPathname == "" {
		return model.Item{
			Name:     "/",
			Pathname: "/",
			IsDir:    true,
		}, nil
	}

	info, err := a.client.StatObject(context.Background(), a.bucket, realPathname, minio.GetObjectOptions{})
	if err != nil {
		return model.Item{}, convertError(fmt.Errorf("unable to stat object: %s", err))
	}

	return convertToItem(info), nil
}

// List items in the storage
func (a App) List(pathname string) ([]model.Item, error) {
	realPathname := getPath(pathname)
	baseRealPathname := path.Base(realPathname)

	objectsCh := a.client.ListObjects(context.Background(), a.bucket, minio.ListObjectsOptions{
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

// WriterTo opens writer for given pathname
func (a App) WriterTo(pathname string) (io.WriteCloser, error) {
	reader, writer := io.Pipe()

	go func() {
		defer func() {
			if closeErr := reader.Close(); closeErr != nil {
				logger.WithField("fn", "s3.WriterTo").WithField("item", pathname).Error("unable to close writer: %s", closeErr)
			}
		}()

		if _, err := a.client.PutObject(context.Background(), a.bucket, getPath(pathname), reader, -1, minio.PutObjectOptions{}); err != nil {
			logger.WithField("fn", "s3.WriterTo").WithField("item", pathname).Error("unable to put object: %s", err)
		}
	}()

	return writer, nil
}

// ReaderFrom reads content from given pathname
func (a App) ReaderFrom(pathname string) (io.ReadSeekCloser, error) {
	object, err := a.client.GetObject(context.Background(), a.bucket, getPath(pathname), minio.GetObjectOptions{})
	if err != nil {
		return nil, convertError(fmt.Errorf("unable to get object: %s", err))
	}

	return object, nil
}

// UpdateDate update date from given value
func (a App) UpdateDate(pathname string, date time.Time) error {
	// TODO

	return nil
}

// Walk browses item recursively
func (a App) Walk(pathname string, walkFn func(model.Item) error) error {
	objectsCh := a.client.ListObjects(context.Background(), a.bucket, minio.ListObjectsOptions{
		Prefix:    getPath(pathname),
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
func (a App) CreateDir(name string) error {
	_, err := a.client.PutObject(context.Background(), a.bucket, model.Dirname(getPath(name)), strings.NewReader(""), 0, minio.PutObjectOptions{})
	if err != nil {
		return convertError(fmt.Errorf("unable to create directory: %s", err))
	}

	return nil
}

// Rename file or directory from storage
func (a App) Rename(oldName, newName string) error {
	oldRoot := getPath(oldName)
	newRoot := getPath(newName)

	return a.Walk(oldRoot, func(item model.Item) error {
		_, err := a.client.CopyObject(context.Background(), minio.CopyDestOptions{
			Bucket: a.bucket,
			Object: strings.Replace(item.Pathname, oldRoot, newRoot, -1),
		}, minio.CopySrcOptions{
			Bucket: a.bucket,
			Object: item.Pathname,
		})
		if err != nil {
			return convertError(err)
		}

		if err = a.client.RemoveObject(context.Background(), a.bucket, item.Pathname, minio.RemoveObjectOptions{}); err != nil {
			return convertError(fmt.Errorf("unable to delete object: %s", err))
		}

		return nil
	})
}

// Remove file or directory from storage
func (a App) Remove(pathname string) error {
	return a.Walk(pathname, func(item model.Item) error {
		if err := a.client.RemoveObject(context.Background(), a.bucket, item.Pathname, minio.RemoveObjectOptions{}); err != nil {
			return convertError(fmt.Errorf("unable to delete object: %s", err))
		}

		return nil
	})
}
