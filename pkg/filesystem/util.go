package filesystem

import (
	"crypto/rand"
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

const tempFlag = os.O_WRONLY | os.O_CREATE | os.O_EXCL

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
		return nil, err
	}

	infos := make([]fs.FileInfo, 0, len(entries))

	for _, entry := range entries {
		info, err := entry.Info()
		if errors.Is(err, fs.ErrNotExist) {
			continue
		}

		if err != nil {
			return nil, fmt.Errorf("read file metadata: %w", err)
		}

		infos = append(infos, info)
	}

	slices.SortFunc(infos, func(left, right fs.FileInfo) int {
		return strings.Compare(sortKey(left), sortKey(right))
	})

	return infos, nil
}

func (a Service) requireDirectory(name string, info fs.FileInfo) error {
	if strings.HasSuffix(name, "/") && !info.IsDir() {
		return a.ConvertError(&fs.PathError{Op: "stat", Path: name, Err: syscall.ENOTDIR})
	}

	return nil
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

func (a Service) createFile(relative string, flags int) (*os.File, error) {
	file, err := a.root.OpenFile(relative, flags, model.RegularFilePerm)
	if model.IsNotExist(a.ConvertError(err)) {
		if err = a.root.MkdirAll(path.Dir(relative), model.DirectoryPerm); err != nil {
			return nil, a.ConvertError(err)
		}

		file, err = a.root.OpenFile(relative, flags, model.RegularFilePerm)
	}

	if err != nil {
		return nil, a.ConvertError(err)
	}

	return file, nil
}

func (a Service) copyTo(writer io.Writer, reader io.Reader) error {
	buf := bufferPool.Get().(*[]byte)
	defer bufferPool.Put(buf)

	// *os.File implements io.ReaderFrom, which makes io.CopyBuffer ignore buf and allocate its own.
	_, err := io.CopyBuffer(writerOnly{Writer: writer}, reader, *buf)

	return a.ConvertError(err)
}

func (a Service) writeDirect(relative string, reader io.Reader) error {
	file, err := a.createFile(relative, model.WriteFlag)
	if err != nil {
		return err
	}

	return errors.Join(a.copyTo(file, reader), file.Close())
}

func (a Service) writeAtomic(relative string, reader io.Reader, existing fs.FileInfo) error {
	tempName := path.Join(path.Dir(relative), ".absto-"+rand.Text()+".tmp")

	temp, err := a.createFile(tempName, tempFlag)
	if err != nil {
		return err
	}

	if existing != nil {
		err = temp.Chmod(existing.Mode().Perm())
	}

	if err == nil {
		err = a.copyTo(temp, reader)
	}

	err = errors.Join(err, temp.Close())

	if err == nil {
		err = a.ConvertError(a.root.Rename(tempName, relative))
	}

	if err != nil {
		if removeErr := a.root.Remove(tempName); removeErr != nil && !errors.Is(removeErr, fs.ErrNotExist) {
			err = errors.Join(err, removeErr)
		}
	}

	return err
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
