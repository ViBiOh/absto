package model

import (
	"context"
	"errors"
	"fmt"
	"io"
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
	writer  io.WriteCloser
	Item
	readdirPosition int
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

func (fi *FileItem) initWriter() error {
	if fi.writer != nil {
		return nil
	}

	var err error

	fi.writer, err = fi.Storage.Writer(context.Background(), fi.Pathname)
	if err != nil {
		return fmt.Errorf("writer: %w", err)
	}

	return nil
}

func (fi *FileItem) Write(p []byte) (n int, err error) {
	if err := fi.initWriter(); err != nil {
		return 0, nil
	}

	return fi.writer.Write(p)
}

func (fi *FileItem) Close() error {
	var err error

	if fi.reader != nil {
		err = fi.reader.Close()
	}

	if fi.writer != nil {
		err = errors.Join(err, fi.writer.Close())
	}

	return err
}

func (fi *FileItem) Readdir(count int) ([]fs.FileInfo, error) {
	if !fi.IsDirValue {
		return nil, os.ErrInvalid
	}

	items, err := fi.Storage.List(context.Background(), fi.Pathname)
	if err != nil {
		return nil, err
	}

	var start, end int

	if count <= 0 {
		end = len(items)
	} else {
		start = fi.readdirPosition
		end = start + count
	}

	output := make([]fs.FileInfo, 0, end-start)
	for ; start < end && start < len(items); start++ {
		output = append(output, items[start])
	}

	return output, nil
}

type Item struct {
	Date       time.Time   `msg:"date" json:"date"`
	ID         string      `msg:"id" json:"id"`
	NameValue  string      `msg:"name" json:"name"`
	Pathname   string      `msg:"pathname" json:"pathname"`
	Extension  string      `msg:"extension" json:"extension"`
	SizeValue  int64       `msg:"size" json:"size"`
	FileMode   os.FileMode `msg:"fileMode" json:"fileMode"`
	IsDirValue bool        `msg:"isDir" json:"isDir"`
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
