package telemetry

import (
	"context"
	"io"
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

func (a App) Info(ctx context.Context, pathname string) (model.Item, error) {
	ctx, span := a.tracer.Start(ctx, "info", trace.WithAttributes(attribute.String("item", pathname)))
	defer span.End()

	output, err := a.storage.Info(ctx, pathname)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return output, err
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

func (a App) ReadFrom(ctx context.Context, pathname string) (io.ReadSeekCloser, error) {
	ctx, span := a.tracer.Start(ctx, "readFrom", trace.WithAttributes(attribute.String("item", pathname)))

	reader, err := a.storage.ReadFrom(ctx, pathname)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return telemetryCloser{
		ReadSeekCloser: reader,
		end:            span.End,
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

func (a App) CreateDir(ctx context.Context, pathname string) error {
	ctx, span := a.tracer.Start(ctx, "createDir", trace.WithAttributes(attribute.String("item", pathname)))
	defer span.End()

	err := a.storage.CreateDir(ctx, pathname)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (a App) Rename(ctx context.Context, oldName, newName string) error {
	ctx, span := a.tracer.Start(ctx, "rename", trace.WithAttributes(attribute.String("item", oldName)), trace.WithAttributes(attribute.String("new", newName)))
	defer span.End()

	err := a.storage.Rename(ctx, oldName, newName)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (a App) Remove(ctx context.Context, pathname string) error {
	ctx, span := a.tracer.Start(ctx, "remove", trace.WithAttributes(attribute.String("item", pathname)))
	defer span.End()

	err := a.storage.Remove(ctx, pathname)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

type telemetryCloser struct {
	io.ReadSeekCloser
	end func(options ...trace.SpanEndOption)
}

func (tc telemetryCloser) Close() error {
	tc.end()

	return tc.ReadSeekCloser.Close()
}

func (a App) ConvertError(err error) error {
	return a.storage.ConvertError(err)
}
