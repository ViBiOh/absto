package telemetry

import (
	"context"
	"io"
	"time"

	"github.com/ViBiOh/absto/pkg/model"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

var _ model.Storage = App{}

// App of the package
type App struct {
	tracer  trace.Tracer
	storage model.Storage
}

// New creates new App from Config
func New(storage model.Storage, tracer trace.Tracer) model.Storage {
	if tracer == nil {
		return storage
	}

	return App{
		storage: storage,
		tracer:  tracer,
	}
}

// Enabled checks that requirements are met
func (a App) Enabled() bool {
	return a.storage.Enabled()
}

// Name of the sotrage
func (a App) Name() string {
	return a.storage.Name()
}

// WithIgnoreFn create a new App with given ignoreFn
func (a App) WithIgnoreFn(ignoreFn func(model.Item) bool) model.Storage {
	return a.storage.WithIgnoreFn(ignoreFn)
}

// Path return full path of pathname
func (a App) Path(pathname string) string {
	return a.storage.Path(pathname)
}

// Info provide metadata about given pathname
func (a App) Info(ctx context.Context, pathname string) (model.Item, error) {
	var span trace.Span
	ctx, span = a.tracer.Start(ctx, "info")
	span.SetAttributes(attribute.String("item", pathname))
	defer span.End()

	return a.storage.Info(ctx, pathname)
}

// List items in the storage
func (a App) List(ctx context.Context, pathname string) ([]model.Item, error) {
	var span trace.Span
	ctx, span = a.tracer.Start(ctx, "list")
	span.SetAttributes(attribute.String("item", pathname))
	defer span.End()

	return a.storage.List(ctx, pathname)
}

// WriteTo with content from reader to pathname
func (a App) WriteTo(ctx context.Context, pathname string, reader io.Reader) error {
	var span trace.Span
	ctx, span = a.tracer.Start(ctx, "writeTo")
	span.SetAttributes(attribute.String("item", pathname))
	defer span.End()

	return a.storage.WriteTo(ctx, pathname, reader)
}

// ReadFrom reads content from given pathname
func (a App) ReadFrom(ctx context.Context, pathname string) (io.ReadSeekCloser, error) {
	var span trace.Span
	ctx, span = a.tracer.Start(ctx, "readFrom")
	span.SetAttributes(attribute.String("item", pathname))
	defer span.End()

	return a.storage.ReadFrom(ctx, pathname)
}

// UpdateDate update date from given value
func (a App) UpdateDate(ctx context.Context, pathname string, date time.Time) error {
	var span trace.Span
	ctx, span = a.tracer.Start(ctx, "info")
	span.SetAttributes(attribute.String("item", pathname))
	defer span.End()

	return a.storage.UpdateDate(ctx, pathname, date)
}

// Walk browses item recursively
func (a App) Walk(ctx context.Context, pathname string, walkFn func(model.Item) error) error {
	var span trace.Span
	ctx, span = a.tracer.Start(ctx, "info")
	span.SetAttributes(attribute.String("item", pathname))
	defer span.End()

	return a.storage.Walk(ctx, pathname, walkFn)
}

// CreateDir container in storage
func (a App) CreateDir(ctx context.Context, name string) error {
	var span trace.Span
	ctx, span = a.tracer.Start(ctx, "createDir")
	span.SetAttributes(attribute.String("item", name))
	defer span.End()

	return a.storage.CreateDir(ctx, name)
}

// Rename file or directory from storage
func (a App) Rename(ctx context.Context, oldName, newName string) error {
	var span trace.Span
	ctx, span = a.tracer.Start(ctx, "rename")
	span.SetAttributes(attribute.String("item", oldName))
	defer span.End()

	return a.storage.Rename(ctx, oldName, newName)
}

// Remove file or directory from storage
func (a App) Remove(ctx context.Context, pathname string) error {
	var span trace.Span
	ctx, span = a.tracer.Start(ctx, "remove")
	span.SetAttributes(attribute.String("item", pathname))
	defer span.End()

	return a.storage.Remove(ctx, pathname)
}
