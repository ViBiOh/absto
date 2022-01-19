package model

import (
	"bytes"
	"io"
	"sync"
	"time"
)

// BufferPool for io.CopyBuffer
var BufferPool = sync.Pool{
	New: func() interface{} {
		return bytes.NewBuffer(make([]byte, 4*1024))
	},
}

// Storage describe action on a storage provider
type Storage interface {
	Enabled() bool
	Name() string
	WithIgnoreFn(ignoreFn func(Item) bool) Storage
	Path(pathname string) string
	Info(pathname string) (Item, error)
	List(pathname string) ([]Item, error)
	WriteTo(pathname string, reader io.Reader) error
	ReadFrom(pathname string) (io.ReadSeekCloser, error)
	Walk(pathname string, walkFn func(Item) error) error
	CreateDir(pathname string) error
	Rename(oldName, newName string) error
	Remove(pathname string) error
	UpdateDate(pathname string, date time.Time) error
}
