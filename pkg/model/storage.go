package model

import (
	"context"
	"io"
	"os"
	"time"
)

const (
	DirectoryPerm   = 0o700
	RegularFilePerm = 0o600

	ReadFlag  = os.O_RDONLY
	WriteFlag = os.O_WRONLY | os.O_CREATE | os.O_TRUNC
)

type WriteOpts struct {
	Size int64
}

type ReadAtSeekCloser interface {
	io.ReadSeekCloser
	io.ReaderAt
}

type Storage interface {
	Enabled() bool
	Close() error
	Path(string) string

	Stat(ctx context.Context, name string) (Item, error)
	Mkdir(ctx context.Context, name string) error
	Rename(ctx context.Context, oldName, newName string) error
	RemoveAll(ctx context.Context, name string) error

	Name() string
	WithIgnoreFn(func(Item) bool) Storage

	List(ctx context.Context, name string) ([]Item, error)
	WriteTo(ctx context.Context, name string, reader io.Reader, opts WriteOpts) error
	ReadFrom(ctx context.Context, name string) (ReadAtSeekCloser, error)
	Walk(ctx context.Context, name string, walkFn func(Item) error) error

	UpdateDate(ctx context.Context, name string, date time.Time) error
	ConvertError(err error) error
}
