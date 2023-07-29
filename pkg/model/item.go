package model

import (
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/zeebo/xxh3"
)

type Item struct {
	Date      time.Time   `json:"date"`
	ID        string      `json:"id"`
	Name      string      `json:"name"`
	Pathname  string      `json:"pathname"`
	Extension string      `json:"extension"`
	Size      int64       `json:"size"`
	FileMode  os.FileMode `json:"fileMode"`
	IsDir     bool        `json:"isDir"`
}

func (i Item) Sys() any {
	return nil
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

func ID(value string) string {
	return strconv.FormatUint(xxh3.HashString(value), 16)
}

func Dirname(name string) string {
	if !strings.HasSuffix(name, "/") {
		return name + "/"
	}
	return name
}
