package filesystem

import (
	"context"
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"sync"
	"syscall"
	"time"

	"github.com/ViBiOh/absto/pkg/model"
)

const Name = "filesystem"

var _ model.Storage = Service{}

var bufferPool = sync.Pool{
	New: func() any {
		buf := make([]byte, 32*1024)
		return &buf
	},
}

type Service struct {
	ignoreFn    func(model.Item) bool
	root        *os.Root
	rootDirname string
	enabled     bool
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

	root, err := os.OpenRoot(rootDirectory)
	if err != nil {
		return Service{}, Service{}.ConvertError(err)
	}

	return Service{
		root:        root,
		rootDirname: info.Name(),
		enabled:     true,
	}, nil
}

func (a Service) Name() string {
	return Name
}

func (a Service) WithIgnoreFn(ignoreFn func(model.Item) bool) model.Storage {
	a.ignoreFn = ignoreFn

	return a
}

func (a Service) Stat(_ context.Context, name string) (model.Item, error) {
	if !a.enabled {
		return model.Item{}, model.ErrNotEnabled
	}

	if err := model.ValidPath(name); err != nil {
		return model.Item{}, err
	}

	return a.stat(name)
}

func (a Service) stat(name string) (model.Item, error) {
	info, err := a.root.Stat(a.relative(name))
	if err != nil {
		return model.Item{}, a.ConvertError(err)
	}

	if strings.HasSuffix(name, "/") && !info.IsDir() {
		return model.Item{}, a.ConvertError(&fs.PathError{Op: "stat", Path: name, Err: syscall.ENOTDIR})
	}

	return a.newItem(a.pathname(name), info), nil
}

func (a Service) List(_ context.Context, name string) ([]model.Item, error) {
	if !a.enabled {
		return nil, model.ErrNotEnabled
	}

	if err := model.ValidPath(name); err != nil {
		return nil, err
	}

	infos, err := a.readDir(a.relative(name))
	if err != nil {
		return nil, a.ConvertError(err)
	}

	dirName := a.pathname(name)

	items := make([]model.Item, 0, len(infos))

	for _, info := range infos {
		item := a.newItem(path.Join(dirName, info.Name()), info)
		if a.ignoreFn != nil && a.ignoreFn(item) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

func (a Service) WriteTo(_ context.Context, name string, reader io.Reader, _ model.WriteOpts) error {
	if !a.enabled {
		return model.ErrNotEnabled
	}

	if err := model.ValidPath(name); err != nil {
		return err
	}

	writer, err := a.getWritableFile(name)
	if model.IsNotExist(err) {
		if err = a.root.MkdirAll(path.Dir(a.relative(name)), model.DirectoryPerm); err != nil {
			return a.ConvertError(err)
		}

		writer, err = a.getWritableFile(name)
	}

	if err != nil {
		return fmt.Errorf("get writable file: %w", err)
	}

	buf := bufferPool.Get().(*[]byte)
	defer bufferPool.Put(buf)

	// *os.File implements io.ReaderFrom, which makes io.CopyBuffer ignore buf and allocate its own.
	if _, err = io.CopyBuffer(writerOnly{Writer: writer}, reader, *buf); err != nil {
		err = a.ConvertError(err)
	}

	return errors.Join(err, writer.Close())
}

func (a Service) ReadFrom(_ context.Context, name string) (model.ReadAtSeekCloser, error) {
	if !a.enabled {
		return nil, model.ErrNotEnabled
	}

	if err := model.ValidPath(name); err != nil {
		return nil, err
	}

	return a.getReadableFile(name)
}

func (a Service) UpdateDate(_ context.Context, name string, date time.Time) error {
	if !a.enabled {
		return model.ErrNotEnabled
	}

	if err := model.ValidPath(name); err != nil {
		return err
	}

	return a.ConvertError(a.root.Chtimes(a.relative(name), date, date))
}

func (a Service) Walk(_ context.Context, name string, walkFn func(model.Item) error) error {
	if !a.enabled {
		return model.ErrNotEnabled
	}

	if err := model.ValidPath(name); err != nil {
		return err
	}

	relative := a.relative(name)

	info, err := a.root.Stat(relative)
	if err != nil {
		return a.ConvertError(err)
	}

	if !info.IsDir() {
		return model.ErrDirExpected
	}

	return a.ConvertError(a.walk(relative, a.pathname(name), info, walkFn))
}

func (a Service) walk(relative, pathname string, info fs.FileInfo, walkFn func(model.Item) error) error {
	item := a.newItem(pathname, info)
	if a.ignoreFn != nil && a.ignoreFn(item) {
		return nil
	}

	if err := walkFn(item); err != nil {
		return err
	}

	if !info.IsDir() {
		return nil
	}

	children, err := a.readDir(relative)
	if err != nil {
		return err
	}

	for _, child := range children {
		name := child.Name()

		if err = a.walk(path.Join(relative, name), path.Join(pathname, name), child, walkFn); err != nil {
			return err
		}
	}

	return nil
}

func (a Service) Mkdir(_ context.Context, name string, perm os.FileMode) error {
	if !a.enabled {
		return model.ErrNotEnabled
	}

	if err := model.ValidPath(name); err != nil {
		return err
	}

	return a.ConvertError(a.root.MkdirAll(a.relative(name), perm))
}

func (a Service) Rename(ctx context.Context, oldName, newName string) error {
	if !a.enabled {
		return model.ErrNotEnabled
	}

	if err := model.ValidPath(oldName); err != nil {
		return err
	}

	if err := model.ValidPath(newName); err != nil {
		return err
	}

	if _, err := a.stat(oldName); err != nil {
		return err
	}

	oldRelative := a.relative(oldName)
	newRelative := a.relative(newName)

	if oldRelative == newRelative {
		return nil
	}

	newDirPath := path.Dir(strings.TrimSuffix(newName, "/"))
	if _, err := a.stat(newDirPath); err != nil {
		if model.IsNotExist(err) {
			if err = a.Mkdir(ctx, newDirPath, model.DirectoryPerm); err != nil {
				return a.ConvertError(err)
			}
		} else {
			return fmt.Errorf("check if new directory exists: %w", err)
		}
	}

	return a.ConvertError(a.root.Rename(oldRelative, newRelative))
}

func (a Service) RemoveAll(_ context.Context, name string) error {
	if !a.enabled {
		return model.ErrNotEnabled
	}

	if err := model.ValidPath(name); err != nil {
		return err
	}

	relativeName := a.relative(name)

	if strings.HasSuffix(name, "/") {
		if info, err := a.root.Stat(relativeName); err == nil && !info.IsDir() {
			return nil
		}
	}

	return a.ConvertError(a.root.RemoveAll(relativeName))
}

func (a Service) ConvertError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, fs.ErrNotExist) || strings.HasSuffix(err.Error(), "not a directory") {
		return model.ErrNotExist(err)
	}

	return err
}
