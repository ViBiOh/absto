package model

import (
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

type Item struct {
	Date       time.Time   `json:"date"      msg:"date"`
	ID         string      `json:"id"        msg:"id"`
	NameValue  string      `json:"name"      msg:"name"`
	Pathname   string      `json:"pathname"  msg:"pathname"`
	Extension  string      `json:"extension" msg:"extension"`
	SizeValue  int64       `json:"size"      msg:"size"`
	FileMode   os.FileMode `json:"fileMode"  msg:"fileMode"`
	IsDirValue bool        `json:"isDir"     msg:"isDir"`
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
