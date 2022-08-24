package model

import (
	"bytes"
	"context"
	"io"
	"sync"
	"time"
)

var BufferPool = sync.Pool{
	New: func() any {
		return bytes.NewBuffer(make([]byte, 4*1024))
	},
}

type WriteOpts struct {
	Size int64
}

type Storage interface {
	Enabled() bool
	Name() string
	WithIgnoreFn(ignoreFn func(Item) bool) Storage
	Path(pathname string) string
	Info(ctx context.Context, pathname string) (Item, error)
	List(ctx context.Context, pathname string) ([]Item, error)
	WriteTo(ctx context.Context, pathname string, reader io.Reader, opts WriteOpts) error
	ReadFrom(ctx context.Context, pathname string) (io.ReadSeekCloser, error)
	Walk(ctx context.Context, pathname string, walkFn func(Item) error) error
	CreateDir(ctx context.Context, pathname string) error
	Rename(ctx context.Context, oldName, newName string) error
	Remove(ctx context.Context, pathname string) error
	UpdateDate(ctx context.Context, pathname string, date time.Time) error
	ConvertError(err error) error
}
