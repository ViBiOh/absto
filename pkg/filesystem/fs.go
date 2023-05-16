package filesystem

import (
	"errors"
	"fmt"
	"io"
	"io/fs"
	"net/http"
	"os"
	"path"
	"path/filepath"
	"sort"
	"strings"

	"github.com/ViBiOh/absto/pkg/model"
)

var (
	_ fs.ReadDirFS   = App{}
	_ fs.ReadFileFS  = App{}
	_ http.File      = &File{}
	_ fs.ReadDirFile = &File{}
	_ io.ReaderAt    = &File{}
	_ io.Seeker      = &File{}
	_ io.Writer      = &File{}
)

type FileInfo struct {
	fs.FileInfo
}

func (fi FileInfo) Type() fs.FileMode {
	return fi.Mode().Type()
}

func (fi FileInfo) Info() (fs.FileInfo, error) {
	return fi, nil
}

type File struct {
	app     App
	reader  model.ReadAtSeekCloser
	writer  io.WriteCloser
	name    string
	lastDir string
}

func (f *File) Stat() (fs.FileInfo, error) {
	info, err := os.Stat(f.name)
	if err != nil {
		return nil, ConvertError(err)
	}

	return info, nil
}

func (f *File) initReader() error {
	if f.reader != nil {
		return nil
	}

	reader, err := f.app.getFile(f.name, os.O_RDONLY)
	if err != nil {
		return ConvertError(err)
	}

	f.reader = reader

	return nil
}

func (f *File) Read(bytes []byte) (int, error) {
	if err := f.initReader(); err != nil {
		return 0, err
	}

	return f.reader.Read(bytes)
}

func (f *File) Seek(offset int64, whence int) (int64, error) {
	if err := f.initReader(); err != nil {
		return 0, err
	}

	return f.reader.Seek(offset, whence)
}

func (f *File) ReadAt(p []byte, off int64) (int, error) {
	if err := f.initReader(); err != nil {
		return 0, err
	}

	return f.reader.ReadAt(p, off)
}

func (f *File) initWriter() error {
	if f.reader != nil {
		return nil
	}

	writer, err := f.app.getWritableFile(f.name)
	if err != nil {
		return ConvertError(err)
	}

	f.writer = writer

	return nil
}

func (f *File) Write(p []byte) (int, error) {
	if err := f.initWriter(); err != nil {
		return 0, err
	}

	return f.writer.Write(p)
}

func (f *File) Close() error {
	var err error

	if f.reader != nil {
		err = f.reader.Close()
	}

	if f.writer != nil {
		err = errors.Join(f.writer.Close())
	}

	return err
}

func (f *File) Readdir(n int) ([]fs.FileInfo, error) {
	entries, err := f.ReadDir(n)
	if err != nil {
		return nil, err
	}

	output := make([]fs.FileInfo, len(entries))

	for i, entry := range entries {
		output[i], err = entry.Info()
		if err != nil {
			return nil, fmt.Errorf("info for `%s`: %w", entry.Name(), err)
		}
	}

	return output, nil
}

func (f *File) ReadDir(n int) ([]fs.DirEntry, error) {
	var output []fs.DirEntry

	err := f.app.walk(f.name, func(info fs.DirEntry) error {
		output = append(output, info)

		return nil
	})
	if err != nil {
		return nil, ConvertError(err)
	}

	sort.Sort(FileByName(output))

	if n <= 0 {
		return output, nil
	}

	if len(f.lastDir) != 0 {
		for i, entry := range output {
			if entry.Name() >= f.lastDir {
				output = output[i+1:]
				break
			}
		}
	}

	output = output[:min(len(output), n)]
	f.lastDir = output[len(output)-1].Name()

	return output, nil
}

func (a App) walk(pathname string, walkFn func(fs.DirEntry) error) error {
	return filepath.Walk(pathname, func(entryName string, info fs.FileInfo, err error) error {
		if err != nil {
			return err
		}

		if pathname == entryName {
			return nil
		}

		if pathname != model.Dirname(path.Dir(entryName)) {
			return filepath.SkipDir
		}

		return walkFn(FileInfo{info})
	})
}

func (a App) Open(name string) (fs.File, error) {
	pathname := a.Path(name)

	if fs.ValidPath(pathname) {
		return nil, model.ErrRelativePath
	}

	return &File{
		name: pathname,
		app:  a,
	}, nil
}

func (a App) OpenFile(name string) (*File, error) {
	pathname := a.Path(name)

	if fs.ValidPath(pathname) {
		return nil, model.ErrRelativePath
	}

	return &File{
		name: pathname,
		app:  a,
	}, nil
}

func (a App) ReadFile(name string) ([]byte, error) {
	return fs.ReadFile(a, name)
}

func (a App) ReadDir(name string) ([]fs.DirEntry, error) {
	file, err := a.Open(name)
	if err != nil {
		return nil, err
	}

	if dir, ok := file.(fs.ReadDirFile); ok {
		return dir.ReadDir(-1)
	}

	return nil, nil
}

// FileByName sort fs.DirEntry by Name
type FileByName []fs.DirEntry

func (a FileByName) Len() int      { return len(a) }
func (a FileByName) Swap(i, j int) { a[i], a[j] = a[j], a[i] }
func (a FileByName) Less(i, j int) bool {
	return a[i].Name() < a[j].Name()
}

func min(a, b int) int {
	if a < b {
		return a
	}
	return b
}

func ConvertError(err error) error {
	if err == nil {
		return nil
	}

	if os.IsNotExist(err) || strings.HasSuffix(err.Error(), "not a directory") {
		return model.ErrNotExist(err)
	}

	return err
}
