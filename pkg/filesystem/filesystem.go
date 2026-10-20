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
	ignoreFn      func(model.Item) bool
	root          *os.Root
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

	root, err := os.OpenRoot(rootDirectory)
	if err != nil {
		return Service{}, Service{}.ConvertError(err)
	}

	return Service{
		root:          root,
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

	info, err := a.root.Stat(a.relative(name))
	if err != nil {
		return model.Item{}, a.ConvertError(err)
	}

	if err = a.requireDirectory(name, info); err != nil {
		return model.Item{}, err
	}

	return a.newItem(a.pathname(name), info), nil
}

func (a Service) List(_ context.Context, name string) ([]model.Item, error) {
	if err := model.ValidPath(name); err != nil {
		return nil, err
	}

	infos, err := a.readDir(a.relative(name))
	if err != nil {
		return nil, a.ConvertError(err)
	}

	items := make([]model.Item, 0, len(infos))

	for _, info := range infos {
		item := a.newItem(path.Join("/", name, info.Name()), info)
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

	if strings.HasSuffix(name, "/") {
		return &fs.PathError{Op: "open", Path: name, Err: syscall.EISDIR}
	}

	relative := a.relative(name)

	existing, err := a.root.Lstat(relative)
	if errors.Is(err, fs.ErrNotExist) {
		return a.writeAtomic(relative, reader, nil)
	}

	if err != nil {
		return a.ConvertError(err)
	}

	switch {
	case existing.IsDir():
		return &fs.PathError{Op: "open", Path: name, Err: syscall.EISDIR}
	case existing.Mode()&fs.ModeSymlink != 0:
		// A rename would replace the link itself instead of writing to its target.
		return a.writeDirect(relative, reader)
	default:
		return a.writeAtomic(relative, reader, existing)
	}
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

	return a.ConvertError(a.root.Chtimes(a.relative(name), date, date))
}

func (a Service) Walk(_ context.Context, name string, walkFn func(model.Item) error) error {
	if err := model.ValidPath(name); err != nil {
		return err
	}

	relative := a.relative(name)

	info, err := a.root.Stat(relative)
	if err != nil {
		return a.ConvertError(err)
	}

	if err = a.requireDirectory(name, info); err != nil {
		return err
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
		if err = a.walk(path.Join(relative, child.Name()), path.Join(pathname, child.Name()), child, walkFn); err != nil {
			return err
		}
	}

	return nil
}

func (a Service) Mkdir(_ context.Context, name string, perm os.FileMode) error {
	if err := model.ValidPath(name); err != nil {
		return err
	}

	return a.ConvertError(a.root.MkdirAll(a.relative(name), perm))
}

func (a Service) Rename(ctx context.Context, oldName, newName string) error {
	if err := model.ValidPath(oldName); err != nil {
		return err
	}

	if err := model.ValidPath(newName); err != nil {
		return err
	}

	if _, err := a.Stat(ctx, oldName); err != nil {
		return err
	}

	if a.relative(oldName) == a.relative(newName) {
		return nil
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

	return a.ConvertError(a.root.Rename(a.relative(oldName), a.relative(newName)))
}

func (a Service) RemoveAll(_ context.Context, name string) error {
	if err := model.ValidPath(name); err != nil {
		return err
	}

	if strings.HasSuffix(name, "/") {
		if info, err := a.root.Stat(a.relative(name)); err == nil && !info.IsDir() {
			return nil
		}
	}

	return a.ConvertError(a.root.RemoveAll(a.relative(name)))
}

func (a Service) ConvertError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
		return model.ErrNotExist(err)
	}

	return err
}
