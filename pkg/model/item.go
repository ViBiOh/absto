package model

import (
	"context"
	"fmt"
	"io/fs"
	"os"
	"path/filepath"
	"strconv"
	"strings"
	"time"

	"github.com/zeebo/xxh3"
)

var (
	_ os.FileInfo = Item{}
	_ fs.FileInfo = Item{}
)

type FileItem struct {
	Storage Storage
	reader  ReadAtSeekCloser
	Item
}

func (fi *FileItem) initReader() error {
	if fi.reader != nil {
		return nil
	}

	var err error

	fi.reader, err = fi.Storage.ReadFrom(context.Background(), fi.Pathname)
	if err != nil {
		return fmt.Errorf("read from: %w", err)
	}

	return nil
}

func (fi *FileItem) Read(p []byte) (int, error) {
	if err := fi.initReader(); err != nil {
		return 0, nil
	}

	return fi.reader.Read(p)
}

func (fi *FileItem) ReadAt(p []byte, off int64) (int, error) {
	if err := fi.initReader(); err != nil {
		return 0, nil
	}

	return fi.reader.ReadAt(p, off)
}

func (fi *FileItem) Seek(offset int64, whence int) (int64, error) {
	if err := fi.initReader(); err != nil {
		return 0, nil
	}

	return fi.reader.Seek(offset, whence)
}

func (fi *FileItem) Close() error {
	if fi.reader == nil {
		return nil
	}

	return fi.reader.Close()
}

type Item struct {
	Date       time.Time   `json:"date"`
	ID         string      `json:"id"`
	NameValue  string      `json:"name"`
	Pathname   string      `json:"pathname"`
	Extension  string      `json:"extension"`
	SizeValue  int64       `json:"size"`
	FileMode   os.FileMode `json:"fileMode"`
	IsDirValue bool        `json:"isDir"`
}

func (i Item) Name() string {
	return i.NameValue
}

func (i Item) Size() int64 {
	return i.SizeValue
}

func (i Item) Mode() os.FileMode {
	return i.FileMode
}

func (i Item) ModTime() time.Time {
	return i.Date
}

func (i Item) IsDir() bool {
	return i.IsDirValue
}

func (i Item) Sys() any {
	return nil
}

func (i Item) String() string {
	var output strings.Builder

	output.WriteString(i.Pathname)
	output.WriteString(strconv.FormatBool(i.IsDirValue))
	output.WriteString(strconv.FormatInt(i.SizeValue, 10))
	output.WriteString(strconv.FormatInt(i.Date.Unix(), 10))

	return output.String()
}

func (i Item) IsZero() bool {
	return len(i.Pathname) == 0
}

func (i Item) Dir() string {
	if i.IsDirValue {
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
