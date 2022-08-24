package filesystem

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ViBiOh/absto/pkg/model"
)

const Name = "fs"

var _ model.Storage = App{}

var ErrRelativePath = errors.New("pathname contains relatives paths")

type App struct {
	ignoreFn      func(model.Item) bool
	rootDirectory string
	rootDirname   string
}

func New(directory string) (App, error) {
	rootDirectory := strings.TrimSuffix(directory, "/")

	if len(rootDirectory) == 0 {
		return App{}, nil
	}

	info, err := os.Stat(rootDirectory)
	if err != nil {
		return App{}, App{}.ConvertError(err)
	}

	if !info.IsDir() {
		return App{}, fmt.Errorf("path %s is not a directory", rootDirectory)
	}

	return App{
		rootDirectory: rootDirectory,
		rootDirname:   info.Name(),
	}, nil
}

func (a App) Enabled() bool {
	return len(a.rootDirectory) != 0
}

func (a App) Name() string {
	return Name
}

func (a App) WithIgnoreFn(ignoreFn func(model.Item) bool) model.Storage {
	a.ignoreFn = ignoreFn

	return a
}

func (a App) Path(pathname string) string {
	if strings.HasPrefix(pathname, "/") {
		return a.rootDirectory + pathname
	}

	return a.rootDirectory + "/" + pathname
}

func (a App) Info(_ context.Context, pathname string) (model.Item, error) {
	if err := checkPathname(pathname); err != nil {
		return model.Item{}, a.ConvertError(err)
	}

	fullpath := a.Path(pathname)

	info, err := os.Stat(fullpath)
	if err != nil {
		return model.Item{}, a.ConvertError(err)
	}

	return convertToItem(a.getRelativePath(fullpath), info), nil
}

func (a App) List(_ context.Context, pathname string) ([]model.Item, error) {
	if err := checkPathname(pathname); err != nil {
		return nil, a.ConvertError(err)
	}

	fullpath := a.Path(pathname)

	files, err := os.ReadDir(fullpath)
	if err != nil {
		return nil, a.ConvertError(err)
	}

	var items []model.Item
	for _, file := range files {
		fileInfo, err := file.Info()
		if err != nil {
			return nil, fmt.Errorf("read file metadata: %w", err)
		}

		item := convertToItem(a.getRelativePath(path.Join(fullpath, file.Name())), fileInfo)
		if a.ignoreFn != nil && a.ignoreFn(item) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

func (a App) WriteTo(_ context.Context, pathname string, reader io.Reader) error {
	if err := checkPathname(pathname); err != nil {
		return a.ConvertError(err)
	}

	writer, err := a.getWritableFile(pathname)
	if err != nil {
		return a.ConvertError(err)
	}

	buffer := model.BufferPool.Get().(*bytes.Buffer)
	defer model.BufferPool.Put(buffer)

	_, err = io.CopyBuffer(writer, reader, buffer.Bytes())
	if err != nil {
		err = a.ConvertError(err)
	}

	return model.HandleClose(writer, err)
}

func (a App) WriteSizedTo(ctx context.Context, pathname string, _ int64, reader io.Reader) error {
	return a.WriteTo(ctx, pathname, reader)
}

func (a App) ReadFrom(_ context.Context, pathname string) (io.ReadSeekCloser, error) {
	if err := checkPathname(pathname); err != nil {
		return nil, a.ConvertError(err)
	}

	output, err := a.getFile(pathname, os.O_RDONLY)
	return output, a.ConvertError(err)
}

func (a App) UpdateDate(_ context.Context, pathname string, date time.Time) error {
	if err := checkPathname(pathname); err != nil {
		return a.ConvertError(err)
	}

	return a.ConvertError(os.Chtimes(a.Path(pathname), date, date))
}

func (a App) Walk(_ context.Context, pathname string, walkFn func(model.Item) error) error {
	pathname = a.Path(pathname)

	return a.ConvertError(filepath.Walk(pathname, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		item := convertToItem(a.getRelativePath(path), info)
		if a.ignoreFn != nil && a.ignoreFn(item) {
			if item.IsDir {
				return filepath.SkipDir
			}
			return nil
		}

		return walkFn(item)
	}))
}

func (a App) CreateDir(_ context.Context, name string) error {
	if err := checkPathname(name); err != nil {
		return a.ConvertError(err)
	}

	return a.ConvertError(os.MkdirAll(a.Path(name), 0o700))
}

func (a App) Rename(ctx context.Context, oldName, newName string) error {
	if err := checkPathname(oldName); err != nil {
		return a.ConvertError(err)
	}

	if err := checkPathname(newName); err != nil {
		return a.ConvertError(err)
	}

	newDirPath := path.Dir(strings.TrimSuffix(newName, "/"))
	if _, err := a.Info(ctx, newDirPath); err != nil {
		if model.IsNotExist(err) {
			if err = a.CreateDir(ctx, newDirPath); err != nil {
				return a.ConvertError(err)
			}
		} else {
			return fmt.Errorf("check if new directory exists: %w", err)
		}
	}

	return a.ConvertError(os.Rename(a.Path(oldName), a.Path(newName)))
}

func (a App) Remove(_ context.Context, pathname string) error {
	if err := checkPathname(pathname); err != nil {
		return a.ConvertError(err)
	}

	return a.ConvertError(os.RemoveAll(a.Path(pathname)))
}

func (a App) ConvertError(err error) error {
	if err == nil {
		return nil
	}

	if os.IsNotExist(err) || strings.HasSuffix(err.Error(), "not a directory") {
		return model.ErrNotExist(err)
	}

	return err
}
