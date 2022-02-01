package model

import (
	"crypto/sha1"
	"encoding/hex"
	"path/filepath"
	"time"
)

// Item describe item on a storage provider
type Item struct {
	Date      time.Time `json:"date"`
	ID        string    `json:"id"`
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

// Sha computes sha1 of given string
func Sha(name string) string {
	hasher := sha1.New()
	_, _ = hasher.Write([]byte(name))
	return hex.EncodeToString(hasher.Sum(nil))
}
