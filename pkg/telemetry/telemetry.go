package telemetry

import (
	"context"
	"io"
	"os"
	"time"

	"github.com/ViBiOh/absto/pkg/model"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
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
	return App{
		storage: a.storage.WithIgnoreFn(ignoreFn),
		tracer:  a.tracer,
	}
}

func (a App) Path(pathname string) string {
	return a.storage.Path(pathname)
}

func (a App) Stat(ctx context.Context, name string) (model.Item, error) {
	ctx, span := a.tracer.Start(ctx, "stat", trace.WithAttributes(attribute.String("name", name)))
	defer span.End()

	output, err := a.storage.Stat(ctx, name)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return output, err
}

func (a App) OpenFile(ctx context.Context, name string, flag int, perm os.FileMode) (*model.FileItem, error) {
	item, err := a.storage.OpenFile(ctx, name, flag, perm)
	if err != nil {
		return nil, err
	}

	item.Storage = a

	return item, nil
}

func (a App) List(ctx context.Context, pathname string) ([]model.Item, error) {
	ctx, span := a.tracer.Start(ctx, "list", trace.WithAttributes(attribute.String("item", pathname)))
	defer span.End()

	output, err := a.storage.List(ctx, pathname)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return output, err
}

func (a App) WriteTo(ctx context.Context, pathname string, reader io.Reader, opts model.WriteOpts) error {
	ctx, span := a.tracer.Start(ctx, "writeTo", trace.WithAttributes(attribute.String("item", pathname)))
	defer span.End()

	err := a.storage.WriteTo(ctx, pathname, reader, opts)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (a App) ReadFrom(ctx context.Context, pathname string) (model.ReadAtSeekCloser, error) {
	ctx, span := a.tracer.Start(ctx, "readFrom", trace.WithAttributes(attribute.String("item", pathname)))

	reader, err := a.storage.ReadFrom(ctx, pathname)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return telemetryCloser{
		ReadAtSeekCloser: reader,
		end:              span.End,
	}, err
}

func (a App) UpdateDate(ctx context.Context, pathname string, date time.Time) error {
	ctx, span := a.tracer.Start(ctx, "updateDate", trace.WithAttributes(attribute.String("item", pathname)))
	defer span.End()

	err := a.storage.UpdateDate(ctx, pathname, date)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (a App) Walk(ctx context.Context, pathname string, walkFn func(model.Item) error) error {
	ctx, span := a.tracer.Start(ctx, "walk", trace.WithAttributes(attribute.String("item", pathname)))
	defer span.End()

	err := a.storage.Walk(ctx, pathname, walkFn)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (a App) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	ctx, span := a.tracer.Start(ctx, "createDir", trace.WithAttributes(attribute.String("name", name)))
	defer span.End()

	err := a.storage.Mkdir(ctx, name, perm)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (a App) Rename(ctx context.Context, oldName, newName string) error {
	ctx, span := a.tracer.Start(ctx, "rename", trace.WithAttributes(attribute.String("oldName", oldName)), trace.WithAttributes(attribute.String("newName", newName)))
	defer span.End()

	err := a.storage.Rename(ctx, oldName, newName)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (a App) RemoveAll(ctx context.Context, name string) error {
	ctx, span := a.tracer.Start(ctx, "remove_all", trace.WithAttributes(attribute.String("name", name)))
	defer span.End()

	err := a.storage.RemoveAll(ctx, name)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

type telemetryCloser struct {
	model.ReadAtSeekCloser
	end func(options ...trace.SpanEndOption)
}

func (tc telemetryCloser) Close() error {
	tc.end()

	return tc.ReadAtSeekCloser.Close()
}

func (a App) ConvertError(err error) error {
	return a.storage.ConvertError(err)
}
