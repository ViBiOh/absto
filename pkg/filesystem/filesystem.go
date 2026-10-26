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
	enabled       bool
}

func New(directory string) (Service, error) {
	rootDirectory := strings.TrimSuffix(directory, "/")

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
		enabled:       true,
	}, nil
}

func (s Service) Name() string {
	return Name
}

func (s Service) WithIgnoreFn(ignoreFn func(model.Item) bool) model.Storage {
	s.ignoreFn = ignoreFn

	return s
}

func (s Service) Enabled() bool {
	return s.enabled
}

func (s Service) Close() error {
	if !s.enabled {
		return model.ErrNotEnabled
	}

	return s.ConvertError(s.root.Close())
}

func (s Service) Path(name string) string {
	return path.Clean(s.rootDirectory) + "/" + s.relative(name)
}

func (s Service) Stat(_ context.Context, name string) (model.Item, error) {
	if !s.enabled {
		return model.Item{}, model.ErrNotEnabled
	}

	if err := model.ValidPath(name); err != nil {
		return model.Item{}, err
	}

	return s.stat(name)
}

func (s Service) stat(name string) (model.Item, error) {
	info, err := s.root.Stat(s.relative(name))
	if err != nil {
		return model.Item{}, s.ConvertError(err)
	}

	if strings.HasSuffix(name, "/") && !info.IsDir() {
		return model.Item{}, s.ConvertError(&fs.PathError{Op: "stat", Path: name, Err: syscall.ENOTDIR})
	}

	return s.newItem(s.pathname(name), info), nil
}

func (s Service) List(_ context.Context, name string) ([]model.Item, error) {
	if !s.enabled {
		return nil, model.ErrNotEnabled
	}

	if err := model.ValidPath(name); err != nil {
		return nil, err
	}

	infos, err := s.readDir(s.relative(name))
	if err != nil {
		return nil, s.ConvertError(err)
	}

	dirName := s.pathname(name)

	items := make([]model.Item, 0, len(infos))

	for _, info := range infos {
		item := s.newItem(path.Join(dirName, info.Name()), info)
		if s.ignoreFn != nil && s.ignoreFn(item) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

func (s Service) WriteTo(_ context.Context, name string, reader io.Reader, _ model.WriteOpts) error {
	if !s.enabled {
		return model.ErrNotEnabled
	}

	if err := model.ValidPath(name); err != nil {
		return err
	}

	writer, err := s.getWritableFile(name)
	if model.IsNotExist(err) {
		if err = s.root.MkdirAll(path.Dir(s.relative(name)), model.DirectoryPerm); err != nil {
			return s.ConvertError(err)
		}

		writer, err = s.getWritableFile(name)
	}

	if err != nil {
		return fmt.Errorf("get writable file: %w", err)
	}

	buf := bufferPool.Get().(*[]byte)
	defer bufferPool.Put(buf)

	// *os.File implements io.ReaderFrom, which makes io.CopyBuffer ignore buf and allocate its own.
	if _, err = io.CopyBuffer(writerOnly{Writer: writer}, reader, *buf); err != nil {
		err = s.ConvertError(err)
	}

	return errors.Join(err, writer.Close())
}

func (s Service) ReadFrom(_ context.Context, name string) (model.ReadAtSeekCloser, error) {
	if !s.enabled {
		return nil, model.ErrNotEnabled
	}

	if err := model.ValidPath(name); err != nil {
		return nil, err
	}

	return s.getReadableFile(name)
}

func (s Service) UpdateDate(_ context.Context, name string, date time.Time) error {
	if !s.enabled {
		return model.ErrNotEnabled
	}

	if err := model.ValidPath(name); err != nil {
		return err
	}

	return s.ConvertError(s.root.Chtimes(s.relative(name), date, date))
}

func (s Service) Walk(_ context.Context, name string, walkFn func(model.Item) error) error {
	if !s.enabled {
		return model.ErrNotEnabled
	}

	if err := model.ValidPath(name); err != nil {
		return err
	}

	relative := s.relative(name)

	info, err := s.root.Stat(relative)
	if err != nil {
		return s.ConvertError(err)
	}

	if !info.IsDir() {
		return model.ErrDirExpected
	}

	return s.ConvertError(s.walk(relative, s.pathname(name), info, walkFn))
}

func (s Service) walk(relative, pathname string, info fs.FileInfo, walkFn func(model.Item) error) error {
	item := s.newItem(pathname, info)
	if s.ignoreFn != nil && s.ignoreFn(item) {
		return nil
	}

	if err := walkFn(item); err != nil {
		return err
	}

	if !info.IsDir() {
		return nil
	}

	children, err := s.readDir(relative)
	if err != nil {
		return err
	}

	for _, child := range children {
		name := child.Name()

		if err = s.walk(path.Join(relative, name), path.Join(pathname, name), child, walkFn); err != nil {
			return err
		}
	}

	return nil
}

func (s Service) Mkdir(_ context.Context, name string) error {
	if !s.enabled {
		return model.ErrNotEnabled
	}

	if err := model.ValidPath(name); err != nil {
		return err
	}

	return s.ConvertError(s.root.MkdirAll(s.relative(name), model.DirectoryPerm))
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

	if _, err := s.stat(oldName); err != nil {
		return err
	}

	oldRelative := s.relative(oldName)
	newRelative := s.relative(newName)

	if oldRelative == newRelative {
		return nil
	}

	newDirPath := path.Dir(strings.TrimSuffix(newName, "/"))
	if _, err := s.stat(newDirPath); err != nil {
		if model.IsNotExist(err) {
			if err = s.Mkdir(ctx, newDirPath); err != nil {
				return s.ConvertError(err)
			}
		} else {
			return fmt.Errorf("check if new directory exists: %w", err)
		}
	}

	return s.ConvertError(s.root.Rename(oldRelative, newRelative))
}

func (s Service) RemoveAll(_ context.Context, name string) error {
	if !s.enabled {
		return model.ErrNotEnabled
	}

	if err := model.ValidPath(name); err != nil {
		return err
	}

	return s.ConvertError(s.root.RemoveAll(s.relative(name)))
}

func (s Service) ConvertError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, fs.ErrNotExist) || errors.Is(err, syscall.ENOTDIR) {
		return model.ErrNotExist(err)
	}

	return err
}
