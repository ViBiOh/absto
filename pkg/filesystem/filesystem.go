package filesystem

import (
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/httputils/v4/pkg/logger"
)

var _ model.Storage = App{}

// ErrRelativePath occurs when path is relative (contains ".."")
var ErrRelativePath = errors.New("pathname contains relatives paths")

// App of the package
type App struct {
	ignoreFn      func(model.Item) bool
	rootDirectory string
	rootDirname   string
}

// New creates new App from Config
func New(directory string) (App, error) {
	rootDirectory := strings.TrimSuffix(directory, "/")

	if len(rootDirectory) == 0 {
		return App{}, errors.New("no directory provided")
	}

	info, err := os.Stat(rootDirectory)
	if err != nil {
		return App{}, convertError(err)
	}

	if !info.IsDir() {
		return App{}, fmt.Errorf("path %s is not a directory", rootDirectory)
	}

	logger.Info("Serving file from %s", rootDirectory)

	return App{
		rootDirectory: rootDirectory,
		rootDirname:   info.Name(),
	}, nil
}

func (a App) path(pathname string) string {
	return path.Join(a.rootDirectory, pathname)
}

// WithIgnoreFn create a new App with given ignoreFn
func (a App) WithIgnoreFn(ignoreFn func(model.Item) bool) model.Storage {
	a.ignoreFn = ignoreFn

	return a
}

// Info provide metadata about given pathname
func (a App) Info(pathname string) (model.Item, error) {
	if err := checkPathname(pathname); err != nil {
		return model.Item{}, convertError(err)
	}

	fullpath := a.path(pathname)

	info, err := os.Stat(fullpath)
	if err != nil {
		return model.Item{}, convertError(err)
	}

	return convertToItem(a.getRelativePath(fullpath), info), nil
}

// List items in the storage
func (a App) List(pathname string) ([]model.Item, error) {
	if err := checkPathname(pathname); err != nil {
		return nil, convertError(err)
	}

	fullpath := a.path(pathname)

	files, err := os.ReadDir(fullpath)
	if err != nil {
		return nil, convertError(err)
	}

	var items []model.Item
	for _, file := range files {
		fileInfo, err := file.Info()
		if err != nil {
			return nil, fmt.Errorf("unable to read file metadata: %s", err)
		}

		item := convertToItem(a.getRelativePath(path.Join(fullpath, file.Name())), fileInfo)
		if a.ignoreFn != nil && a.ignoreFn(item) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

// WriterTo opens writer for given pathname
func (a App) WriterTo(pathname string) (io.WriteCloser, error) {
	if err := checkPathname(pathname); err != nil {
		return nil, convertError(err)
	}

	writer, err := a.getWritableFile(pathname)
	return writer, convertError(err)
}

// ReaderFrom reads content from given pathname
func (a App) ReaderFrom(pathname string) (io.ReadSeekCloser, error) {
	if err := checkPathname(pathname); err != nil {
		return nil, convertError(err)
	}

	output, err := a.getFile(pathname, os.O_RDONLY)
	return output, convertError(err)
}

// UpdateDate update date from given value
func (a App) UpdateDate(pathname string, date time.Time) error {
	if err := checkPathname(pathname); err != nil {
		return convertError(err)
	}

	return convertError(os.Chtimes(a.path(pathname), date, date))
}

// Walk browses item recursively
func (a App) Walk(pathname string, walkFn func(model.Item) error) error {
	pathname = path.Join(a.rootDirectory, pathname)

	return convertError(filepath.Walk(pathname, func(path string, info os.FileInfo, err error) error {
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

// CreateDir container in storage
func (a App) CreateDir(name string) error {
	if err := checkPathname(name); err != nil {
		return convertError(err)
	}

	return convertError(os.MkdirAll(a.path(name), 0o700))
}

// Rename file or directory from storage
func (a App) Rename(oldName, newName string) error {
	if err := checkPathname(oldName); err != nil {
		return convertError(err)
	}

	if err := checkPathname(newName); err != nil {
		return convertError(err)
	}

	newDirPath := path.Dir(strings.TrimSuffix(newName, "/"))
	if _, err := a.Info(newDirPath); err != nil {
		if model.IsNotExist(err) {
			if err = a.CreateDir(newDirPath); err != nil {
				return convertError(err)
			}
		} else {
			return fmt.Errorf("unable to check if new directory exists: %s", err)
		}
	}

	return convertError(os.Rename(a.path(oldName), a.path(newName)))
}

// Remove file or directory from storage
func (a App) Remove(pathname string) error {
	if err := checkPathname(pathname); err != nil {
		return convertError(err)
	}

	return convertError(os.RemoveAll(a.path(pathname)))
}
