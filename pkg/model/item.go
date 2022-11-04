package model

import (
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/zeebo/xxh3"
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

func (i Item) String() string {
	var output strings.Builder

	output.WriteString(i.Pathname)
	output.WriteString(strconv.FormatBool(i.IsDir))
	output.WriteString(strconv.FormatInt(i.Size, 10))
	output.WriteString(strconv.FormatInt(i.Date.Unix(), 10))

	return output.String()
}

func (i Item) IsZero() bool {
	return len(i.Pathname) == 0
}

func (i Item) Dir() string {
	if i.IsDir {
		return i.Pathname
	}

	return Dirname(filepath.Dir(i.Pathname))
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
	return strconv.FormatUint(xxh3.HashString(name), 16)
}
