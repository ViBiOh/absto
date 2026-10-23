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

func (s Service) relative(name string) string {
	relative := strings.TrimPrefix(path.Clean("/"+name), "/")
	if relative == "" {
		return "."
	}

	return relative
}

func (s Service) pathname(name string) string {
	return path.Clean("/" + name)
}

func (s Service) readDir(relative string) ([]fs.FileInfo, error) {
	entries, err := fs.ReadDir(s.root.FS(), relative)
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

func (s Service) newItem(pathname string, info fs.FileInfo) model.Item {
	item := convertToItem(pathname, info)

	if info.Name() == "." {
		item.NameValue = s.rootDirname
	}

	return item
}

func (s Service) getReadableFile(filename string) (model.ReadAtSeekCloser, error) {
	file, err := s.getFile(filename, model.ReadFlag)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (s Service) getWritableFile(filename string) (io.WriteCloser, error) {
	file, err := s.getFile(filename, model.WriteFlag)
	if err != nil {
		return nil, err
	}

	return file, nil
}

func (s Service) getFile(filename string, flags int) (*os.File, error) {
	if strings.HasSuffix(filename, "/") {
		return nil, &fs.PathError{Op: "open", Path: filename, Err: syscall.EISDIR}
	}

	relativeName := s.relative(filename)

	file, err := s.root.OpenFile(relativeName, flags, model.RegularFilePerm)
	if err != nil {
		return nil, s.ConvertError(err)
	}

	if flags == model.ReadFlag {
		if info, err := file.Stat(); err != nil {
			return nil, errors.Join(s.ConvertError(err), file.Close())
		} else if info.IsDir() {
			return nil, errors.Join(&fs.PathError{Op: "open", Path: filename, Err: syscall.EISDIR}, file.Close())
		}
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
