package model

import (
	"io"
	"time"
)

// Storage describe action on a storage provider
type Storage interface {
	WithIgnoreFn(ignoreFn func(Item) bool) Storage
	Info(pathname string) (Item, error)
	List(pathname string) ([]Item, error)
	WriterTo(pathname string) (io.WriteCloser, error)
	ReaderFrom(pathname string) (io.ReadSeekCloser, error)
	Walk(pathname string, walkFn func(Item) error) error
	CreateDir(pathname string) error
	Rename(oldName, newName string) error
	Remove(pathname string) error
	UpdateDate(pathname string, date time.Time) error
}
