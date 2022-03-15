package filesystem

import (
	"bytes"
	"context"
	"errors"
	"fmt"
	"io"
	"os"
	"path"
	"path/filepath"
	"strings"
	"time"

	"github.com/ViBiOh/absto/pkg/model"
	"go.opentelemetry.io/otel/trace"
)

// Name of the storage implementation
const Name = "fs"

var _ model.Storage = App{}

// ErrRelativePath occurs when path is relative (contains ".."")
var ErrRelativePath = errors.New("pathname contains relatives paths")

// Option for configuring client
type Option func(App) App

// WithTracer add open telemetry tracer to context
func WithTracer(tracer trace.Tracer) func(App) App {
	return func(a App) App {
		a.tracer = tracer
		return a
	}
}

// App of the package
type App struct {
	tracer        trace.Tracer
	ignoreFn      func(model.Item) bool
	rootDirectory string
	rootDirname   string
}

// New creates new App from Config
func New(directory string, opts ...Option) (App, error) {
	rootDirectory := strings.TrimSuffix(directory, "/")

	if len(rootDirectory) == 0 {
		return App{}, nil
	}

	info, err := os.Stat(rootDirectory)
	if err != nil {
		return App{}, convertError(err)
	}

	if !info.IsDir() {
		return App{}, fmt.Errorf("path %s is not a directory", rootDirectory)
	}

	app := App{
		rootDirectory: rootDirectory,
		rootDirname:   info.Name(),
	}

	for _, opt := range opts {
		app = opt(app)
	}

	return app, nil
}

// Enabled checks that requirements are met
func (a App) Enabled() bool {
	return len(a.rootDirectory) != 0
}

// Name of the sotrage
func (a App) Name() string {
	return Name
}

// WithIgnoreFn create a new App with given ignoreFn
func (a App) WithIgnoreFn(ignoreFn func(model.Item) bool) model.Storage {
	a.ignoreFn = ignoreFn

	return a
}

// Path return full path of pathname
func (a App) Path(pathname string) string {
	if strings.HasPrefix(pathname, "/") {
		return a.rootDirectory + pathname
	}

	return a.rootDirectory + "/" + pathname
}

// Info provide metadata about given pathname
func (a App) Info(ctx context.Context, pathname string) (model.Item, error) {
	if a.tracer != nil {
		var span trace.Span
		_, span = a.tracer.Start(ctx, "info")
		defer span.End()
	}

	if err := checkPathname(pathname); err != nil {
		return model.Item{}, convertError(err)
	}

	fullpath := a.Path(pathname)

	info, err := os.Stat(fullpath)
	if err != nil {
		return model.Item{}, convertError(err)
	}

	return convertToItem(a.getRelativePath(fullpath), info), nil
}

// List items in the storage
func (a App) List(ctx context.Context, pathname string) ([]model.Item, error) {
	if a.tracer != nil {
		var span trace.Span
		_, span = a.tracer.Start(ctx, "list")
		defer span.End()
	}

	if err := checkPathname(pathname); err != nil {
		return nil, convertError(err)
	}

	fullpath := a.Path(pathname)

	files, err := os.ReadDir(fullpath)
	if err != nil {
		return nil, convertError(err)
	}

	var items []model.Item
	for _, file := range files {
		fileInfo, err := file.Info()
		if err != nil {
			return nil, fmt.Errorf("unable to read file metadata: %s", err)
		}

		item := convertToItem(a.getRelativePath(path.Join(fullpath, file.Name())), fileInfo)
		if a.ignoreFn != nil && a.ignoreFn(item) {
			continue
		}

		items = append(items, item)
	}

	return items, nil
}

// WriteTo with content from reader to pathname
func (a App) WriteTo(ctx context.Context, pathname string, reader io.Reader) error {
	if a.tracer != nil {
		var span trace.Span
		_, span = a.tracer.Start(ctx, "writeTo")
		defer span.End()
	}

	if err := checkPathname(pathname); err != nil {
		return convertError(err)
	}

	writer, err := a.getWritableFile(pathname)
	if err != nil {
		return convertError(err)
	}

	buffer := model.BufferPool.Get().(*bytes.Buffer)
	defer model.BufferPool.Put(buffer)

	_, err = io.CopyBuffer(writer, reader, buffer.Bytes())
	if err != nil {
		err = convertError(err)
	}

	return model.HandleClose(writer, err)
}

// ReadFrom reads content from given pathname
func (a App) ReadFrom(ctx context.Context, pathname string) (io.ReadSeekCloser, error) {
	if a.tracer != nil {
		var span trace.Span
		_, span = a.tracer.Start(ctx, "readFrom")
		defer span.End()
	}

	if err := checkPathname(pathname); err != nil {
		return nil, convertError(err)
	}

	output, err := a.getFile(pathname, os.O_RDONLY)
	return output, convertError(err)
}

// UpdateDate update date from given value
func (a App) UpdateDate(ctx context.Context, pathname string, date time.Time) error {
	if a.tracer != nil {
		var span trace.Span
		_, span = a.tracer.Start(ctx, "updateDate")
		defer span.End()
	}

	if err := checkPathname(pathname); err != nil {
		return convertError(err)
	}

	return convertError(os.Chtimes(a.Path(pathname), date, date))
}

// Walk browses item recursively
func (a App) Walk(ctx context.Context, pathname string, walkFn func(model.Item) error) error {
	if a.tracer != nil {
		var span trace.Span
		_, span = a.tracer.Start(ctx, "walk")
		defer span.End()
	}

	pathname = a.Path(pathname)

	return convertError(filepath.Walk(pathname, func(path string, info os.FileInfo, err error) error {
		if err != nil {
			return err
		}

		item := convertToItem(a.getRelativePath(path), info)
		if a.ignoreFn != nil && a.ignoreFn(item) {
			if item.IsDir {
				return filepath.SkipDir
			}
			return nil
		}

		return walkFn(item)
	}))
}

// CreateDir container in storage
func (a App) CreateDir(ctx context.Context, name string) error {
	if a.tracer != nil {
		var span trace.Span
		_, span = a.tracer.Start(ctx, "createDir")
		defer span.End()
	}

	if err := checkPathname(name); err != nil {
		return convertError(err)
	}

	return convertError(os.MkdirAll(a.Path(name), 0o700))
}

// Rename file or directory from storage
func (a App) Rename(ctx context.Context, oldName, newName string) error {
	if a.tracer != nil {
		var span trace.Span
		_, span = a.tracer.Start(ctx, "rename")
		defer span.End()
	}

	if err := checkPathname(oldName); err != nil {
		return convertError(err)
	}

	if err := checkPathname(newName); err != nil {
		return convertError(err)
	}

	newDirPath := path.Dir(strings.TrimSuffix(newName, "/"))
	if _, err := a.Info(ctx, newDirPath); err != nil {
		if model.IsNotExist(err) {
			if err = a.CreateDir(ctx, newDirPath); err != nil {
				return convertError(err)
			}
		} else {
			return fmt.Errorf("unable to check if new directory exists: %s", err)
		}
	}

	return convertError(os.Rename(a.Path(oldName), a.Path(newName)))
}

// Remove file or directory from storage
func (a App) Remove(ctx context.Context, pathname string) error {
	if a.tracer != nil {
		var span trace.Span
		_, span = a.tracer.Start(ctx, "remove")
		defer span.End()
	}

	if err := checkPathname(pathname); err != nil {
		return convertError(err)
	}

	return convertError(os.RemoveAll(a.Path(pathname)))
}
