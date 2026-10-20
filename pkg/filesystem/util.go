package filesystem

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"os"
	"path"
	"slices"
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
	return path.Clean("/" + name)
}

func (a Service) readDir(relative string) ([]fs.FileInfo, error) {
	entries, err := fs.ReadDir(a.root.FS(), relative)
	if err != nil {
		return nil, fmt.Errorf("read dir: %w", err)
	}

	infos := make([]fs.FileInfo, 0, len(entries))

	for _, entry := range entries {
		info, err := entry.Info()
		if err != nil {
			if errors.Is(err, fs.ErrNotExist) {
				continue
			}

			return nil, fmt.Errorf("read file metadata: %w", err)
		}

		infos = append(infos, info)
	}

	slices.SortFunc(infos, func(left, right fs.FileInfo) int {
		return strings.Compare(sortKey(left), sortKey(right))
	})

	return infos, nil
}

func sortKey(info fs.FileInfo) string {
	if info.IsDir() {
		return info.Name() + "/"
	}

	return info.Name()
}

func (a Service) newItem(pathname string, info fs.FileInfo) model.Item {
	item := convertToItem(pathname, info)

	if info.Name() == "." {
		item.NameValue = a.rootDirname
	}

	return item
}

func (a Service) getReadableFile(filename string) (model.ReadAtSeekCloser, error) {
	return a.getFile(filename, model.ReadFlag)
}

func (a Service) getWritableFile(filename string) (io.WriteCloser, error) {
	return a.getFile(filename, model.WriteFlag)
}

func (a Service) getFile(filename string, flags int) (*os.File, error) {
	relativeName := a.relative(filename)

	if strings.HasSuffix(relativeName, "/") {
		return nil, &fs.PathError{Op: "open", Path: filename, Err: syscall.EISDIR}
	}

	file, err := a.root.OpenFile(relativeName, flags, model.RegularFilePerm)
	if err != nil {
		return nil, a.ConvertError(err)
	}

	return file, nil
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
