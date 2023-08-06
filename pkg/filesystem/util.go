package filesystem

import (
	"io"
	"io/fs"
	"os"
	"path"
	"strings"

	"github.com/ViBiOh/absto/pkg/model"
)

func (a App) getRelativePath(pathname string) string {
	return strings.TrimPrefix(pathname, a.rootDirectory)
}

func (a App) getFile(filename string, flags int) (*os.File, error) {
	file, err := os.OpenFile(a.Path(filename), flags, getMode(filename))
	return file, a.ConvertError(err)
}

func (a App) getReadableFile(filename string) (model.ReadAtSeekCloser, error) {
	return a.getFile(filename, os.O_RDONLY)
}

func (a App) getWritableFile(filename string) (io.WriteCloser, error) {
	return a.getFile(filename, os.O_RDWR|os.O_CREATE|os.O_TRUNC)
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
