package model

import (
	"io"
	"time"
)

// Storage describe action on a storage provider
type Storage interface {
	Enabled() bool
	Name() string
	WithIgnoreFn(ignoreFn func(Item) bool) Storage
	Path(pathname string) string
	Info(pathname string) (Item, error)
	List(pathname string) ([]Item, error)
	WriterTo(pathname string) (io.Writer, Closer, error)
	ReaderFrom(pathname string) (io.ReadSeekCloser, error)
	Walk(pathname string, walkFn func(Item) error) error
	CreateDir(pathname string) error
	Rename(oldName, newName string) error
	Remove(pathname string) error
	UpdateDate(pathname string, date time.Time) error
}

// Closer closes writer
type Closer func() error

// NoopCloser does nothing
var NoopCloser = func() error {
	return nil
}
