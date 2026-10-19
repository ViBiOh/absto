package filesystem

import (
	"io"
	"io/fs"
	"os"
	"path"
	"strings"
	"syscall"

	"github.com/ViBiOh/absto/pkg/model"
)

type writerOnly struct {
	io.Writer
}

func (a Service) relative(name string) string {
	relative := strings.TrimPrefix(path.Clean("/"+name), "/")
	if relative == "" {
		return "."
	}

	return relative
}

func (a Service) pathname(name string) string {
	if strings.HasPrefix(name, "/") {
		return name
	}

	return "/" + name
}

func (a Service) newItem(pathname string, info fs.FileInfo) model.Item {
	item := convertToItem(pathname, info)

	if info.Name() == "." {
		item.NameValue = a.rootDirname
	}

	return item
}

func (a Service) getFile(filename string, flags int) (*os.File, error) {
	file, err := a.root.OpenFile(a.relative(filename), flags, getMode(filename))
	if err != nil {
		return nil, a.ConvertError(err)
	}

	return file, nil
}

func (a Service) getReadableFile(filename string) (model.ReadAtSeekCloser, error) {
	file, err := a.getFile(filename, model.ReadFlag)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (a Service) getWritableFile(filename string) (io.WriteCloser, error) {
	if strings.HasSuffix(filename, "/") {
		return nil, &fs.PathError{Op: "open", Path: filename, Err: syscall.EISDIR}
	}

	file, err := a.getFile(filename, model.WriteFlag)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func getMode(name string) os.FileMode {
	if strings.HasSuffix(name, "/") {
		return model.DirectoryPerm
	}

	return model.RegularFilePerm
}

func convertToItem(pathname string, info fs.FileInfo) model.Item {
	name := info.Name()

	item := model.Item{
		ID:         model.ID(pathname),
		NameValue:  name,
		Pathname:   pathname,
		IsDirValue: info.IsDir(),
		Date:       info.ModTime(),
		FileMode:   info.Mode(),
	}

	if !item.IsDir() {
		item.Extension = strings.ToLower(path.Ext(name))
		item.SizeValue = info.Size()
	}

	return item
}
