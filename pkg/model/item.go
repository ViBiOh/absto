package model

import (
	"path/filepath"
	"time"
)

// Item describe item on a storage provider
type Item struct {
	Date      time.Time `json:"date"`
	Name      string    `json:"name"`
	Pathname  string    `json:"pathname"`
	Extension string    `json:"extension"`
	IsDir     bool      `json:"isDir"`
	Size      int64     `json:"size"`
}

// Dir return the nearest directory (self of parent)
func (s Item) Dir() string {
	if s.IsDir {
		return s.Pathname
	}

	return filepath.Dir(s.Pathname)
}
