package model

import (
	"context"
	"io"
	"io/fs"
	"os"
	"time"
)

const (
	DirectoryPerm   = 0o700
	RegularFilePerm = 0o600

	ReadFlag  = os.O_RDONLY
	WriteFlag = os.O_RDWR | os.O_CREATE | os.O_TRUNC
)

type WriteOpts struct {
	Size int64
}

type ReadAtSeekCloser interface {
	io.ReadSeekCloser
	io.ReaderAt
}

type File interface {
	ReadAtSeekCloser
	io.Writer
	Readdir(int) ([]fs.FileInfo, error)
	Stat() (fs.FileInfo, error)
}

type Storage interface {
	Stat(ctx context.Context, name string) (Item, error)
	Mkdir(ctx context.Context, name string, perm os.FileMode) error
	Rename(ctx context.Context, oldName, newName string) error
	RemoveAll(ctx context.Context, name string) error

	Enabled() bool
	Name() string
	WithIgnoreFn(ignoreFn func(Item) bool) Storage
	Path(name string) string

	List(ctx context.Context, name string) ([]Item, error)
	WriteTo(ctx context.Context, name string, reader io.Reader, opts WriteOpts) error
	ReadFrom(ctx context.Context, name string) (ReadAtSeekCloser, error)
	Walk(ctx context.Context, name string, walkFn func(Item) error) error

	UpdateDate(ctx context.Context, name string, date time.Time) error
	ConvertError(err error) error
}
