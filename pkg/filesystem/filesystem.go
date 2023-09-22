package filesystem

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"path/filepath"
	"strings"
	"sync"
	"time"

	"github.com/ViBiOh/absto/pkg/model"
)

const Name = "filesystem"

var _ model.Storage = Service{}

var bufferPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 32*1024))
	},
}

type Service struct {
	ignoreFn      func(model.Item) bool
	rootDirectory string
	rootDirname   string
}

func New(directory string) (Service, error) {
	rootDirectory := strings.TrimSuffix(directory, "/")

	if len(rootDirectory) == 0 {
		return Service{}, nil
	}

	info, err := os.Stat(rootDirectory)
	if err != nil {
		return Service{}, Service{}.ConvertError(err)
	}

	if !info.IsDir() {
		return Service{}, fmt.Errorf("path %s is not a directory", rootDirectory)
	}

	return Service{
		rootDirectory: rootDirectory,
		rootDirname:   info.Name(),
	}, nil
}

func (a Service) Enabled() bool {
	return len(a.rootDirectory) != 0
}

func (a Service) Name() string {
	return Name
}

func (a Service) WithIgnoreFn(ignoreFn func(model.Item) bool) model.Storage {
	a.ignoreFn = ignoreFn

	return a
}

func (a Service) Path(name string) string {
	if strings.HasPrefix(name, "/") {
		return a.rootDirectory + name
	}

	return a.rootDirectory + "/" + name
}

func (a Service) Stat(_ context.Context, name string) (model.Item, error) {
	if err := model.ValidPath(name); err != nil {
		return model.Item{}, err
	}

	fullpath := a.Path(name)

	info, err := os.Stat(fullpath)
	if err != nil {
		return model.Item{}, a.ConvertError(err)
	}

	return convertToItem(a.getRelativePath(fullpath), info), nil
}

func (a Service) List(_ context.Context, name string) ([]model.Item, error) {
	if err := model.ValidPath(name); err != nil {
		return nil, err
	}

	fullpath := a.Path(name)

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

func (a Service) WriteTo(_ context.Context, name string, reader io.Reader, _ model.WriteOpts) error {
	if err := model.ValidPath(name); err != nil {
		return err
	}

	writer, err := a.getWritableFile(name)
	if err != nil {
		return err
	}

	buffer := bufferPool.Get().(*bytes.Buffer)
	defer bufferPool.Put(buffer)

	if _, err = io.CopyBuffer(writer, reader, buffer.Bytes()); err != nil {
		err = a.ConvertError(err)
	}

	return errors.Join(err, writer.Close())
}

func (a Service) ReadFrom(_ context.Context, name string) (model.ReadAtSeekCloser, error) {
	if err := model.ValidPath(name); err != nil {
		return nil, err
	}

	return a.getReadableFile(name)
}

func (a Service) UpdateDate(_ context.Context, name string, date time.Time) error {
	if err := model.ValidPath(name); err != nil {
		return err
	}

	return a.ConvertError(os.Chtimes(a.Path(name), date, date))
}

func (a Service) Walk(_ context.Context, name string, walkFn func(model.Item) error) error {
	name = a.Path(name)

	return a.ConvertError(filepath.Walk(name, func(path string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		item := convertToItem(a.getRelativePath(path), info)
		if a.ignoreFn != nil && a.ignoreFn(item) {
			if item.IsDir() {
				return filepath.SkipDir
			}
			return nil
		}

		return walkFn(item)
	}))
}

func (a Service) Mkdir(_ context.Context, name string, perm os.FileMode) error {
	if err := model.ValidPath(name); err != nil {
		return err
	}

	return a.ConvertError(os.MkdirAll(a.Path(name), perm))
}

func (a Service) Rename(ctx context.Context, oldName, newName string) error {
	if err := model.ValidPath(oldName); err != nil {
		return err
	}

	if err := model.ValidPath(newName); err != nil {
		return err
	}

	newDirPath := path.Dir(strings.TrimSuffix(newName, "/"))
	if _, err := a.Stat(ctx, newDirPath); err != nil {
		if model.IsNotExist(err) {
			if err = a.Mkdir(ctx, newDirPath, model.DirectoryPerm); err != nil {
				return a.ConvertError(err)
			}
		} else {
			return fmt.Errorf("check if new directory exists: %w", err)
		}
	}

	return a.ConvertError(os.Rename(a.Path(oldName), a.Path(newName)))
}

func (a Service) RemoveAll(_ context.Context, name string) error {
	if err := model.ValidPath(name); err != nil {
		return err
	}

	return a.ConvertError(os.RemoveAll(a.Path(name)))
}

func (a Service) ConvertError(err error) error {
	if err == nil {
		return nil
	}

	if os.IsNotExist(err) || strings.HasSuffix(err.Error(), "not a directory") {
		return model.ErrNotExist(err)
	}

	return err
}
