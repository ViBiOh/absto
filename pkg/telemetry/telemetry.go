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

type App struct {
	tracer  trace.Tracer
	storage model.Storage
}

func New(storage model.Storage, tracer trace.Tracer) model.Storage {
	if tracer == nil {
		return storage
	}

	return App{
		storage: storage,
		tracer:  tracer,
	}
}

func (a App) Enabled() bool {
	return a.storage.Enabled()
}

func (a App) Name() string {
	return a.storage.Name()
}

func (a App) WithIgnoreFn(ignoreFn func(model.Item) bool) model.Storage {
	return a.storage.WithIgnoreFn(ignoreFn)
}

func (a App) Path(pathname string) string {
	return a.storage.Path(pathname)
}

func (a App) Info(ctx context.Context, pathname string) (model.Item, error) {
	ctx, span := a.tracer.Start(ctx, "info", trace.WithAttributes(attribute.String("item", pathname)))
	defer span.End()

	return a.storage.Info(ctx, pathname)
}

func (a App) List(ctx context.Context, pathname string) ([]model.Item, error) {
	ctx, span := a.tracer.Start(ctx, "list", trace.WithAttributes(attribute.String("item", pathname)))
	defer span.End()

	return a.storage.List(ctx, pathname)
}

func (a App) WriteTo(ctx context.Context, pathname string, reader io.Reader) error {
	ctx, span := a.tracer.Start(ctx, "writeTo", trace.WithAttributes(attribute.String("item", pathname)))
	defer span.End()

	return a.storage.WriteTo(ctx, pathname, reader)
}

func (a App) WriteSizedTo(ctx context.Context, pathname string, size int64, reader io.Reader) error {
	ctx, span := a.tracer.Start(ctx, "writeTo", trace.WithAttributes(attribute.String("item", pathname)), trace.WithAttributes(attribute.Int64("size", size)))
	defer span.End()

	return a.storage.WriteSizedTo(ctx, pathname, size, reader)
}

func (a App) ReadFrom(ctx context.Context, pathname string) (io.ReadSeekCloser, error) {
	ctx, span := a.tracer.Start(ctx, "readFrom", trace.WithAttributes(attribute.String("item", pathname)))
	defer span.End()

	return a.storage.ReadFrom(ctx, pathname)
}

func (a App) UpdateDate(ctx context.Context, pathname string, date time.Time) error {
	ctx, span := a.tracer.Start(ctx, "updateDate", trace.WithAttributes(attribute.String("item", pathname)))
	defer span.End()

	return a.storage.UpdateDate(ctx, pathname, date)
}

func (a App) Walk(ctx context.Context, pathname string, walkFn func(model.Item) error) error {
	ctx, span := a.tracer.Start(ctx, "walk", trace.WithAttributes(attribute.String("item", pathname)))
	defer span.End()

	return a.storage.Walk(ctx, pathname, walkFn)
}

func (a App) CreateDir(ctx context.Context, pathname string) error {
	ctx, span := a.tracer.Start(ctx, "createDir", trace.WithAttributes(attribute.String("item", pathname)))
	defer span.End()

	return a.storage.CreateDir(ctx, pathname)
}

func (a App) Rename(ctx context.Context, oldName, newName string) error {
	ctx, span := a.tracer.Start(ctx, "rename", trace.WithAttributes(attribute.String("item", oldName)), trace.WithAttributes(attribute.String("new", newName)))
	defer span.End()

	return a.storage.Rename(ctx, oldName, newName)
}

func (a App) Remove(ctx context.Context, pathname string) error {
	ctx, span := a.tracer.Start(ctx, "remove", trace.WithAttributes(attribute.String("item", pathname)))
	defer span.End()

	return a.storage.Remove(ctx, pathname)
}

func (a App) ConvertError(err error) error {
	return a.storage.ConvertError(err)
}
