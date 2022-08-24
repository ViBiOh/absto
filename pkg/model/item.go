package model

import (
	"crypto/sha256"
	"encoding/hex"
	"path/filepath"
	"strings"
	"time"
)

type Item struct {
	Date      time.Time `json:"date"`
	ID        string    `json:"id"`
	Name      string    `json:"name"`
	Pathname  string    `json:"pathname"`
	Extension string    `json:"extension"`
	IsDir     bool      `json:"isDir"`
	Size      int64     `json:"size"`
}

func (s Item) Dir() string {
	if s.IsDir {
		return s.Pathname
	}

	return Dirname(filepath.Dir(s.Pathname))
}

func ID(pathname string) string {
	return sha(pathname)
}

func Dirname(name string) string {
	if !strings.HasSuffix(name, "/") {
		return name + "/"
	}
	return name
}

func sha(name string) string {
	hasher := sha256.New()
	_, _ = hasher.Write([]byte(name))
	return hex.EncodeToString(hasher.Sum(nil))
}
