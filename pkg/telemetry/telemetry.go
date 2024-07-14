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

var _ model.Storage = Service{}

type Service struct {
	tracer  trace.Tracer
	storage model.Storage
}

func New(storage model.Storage, tracerProvider trace.TracerProvider) model.Storage {
	if tracerProvider == nil {
		return storage
	}

	return Service{
		storage: storage,
		tracer:  tracerProvider.Tracer("absto"),
	}
}

func (a Service) Enabled() bool {
	return a.storage.Enabled()
}

func (a Service) Name() string {
	return a.storage.Name()
}

func (a Service) WithIgnoreFn(ignoreFn func(model.Item) bool) model.Storage {
	return Service{
		storage: a.storage.WithIgnoreFn(ignoreFn),
		tracer:  a.tracer,
	}
}

func (a Service) Path(name string) string {
	return a.storage.Path(name)
}

func (a Service) Stat(ctx context.Context, name string) (model.Item, error) {
	ctx, span := a.tracer.Start(ctx, "stat", trace.WithAttributes(attribute.String("name", name)))
	defer span.End()

	output, err := a.storage.Stat(ctx, name)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return output, err
}

func (a Service) List(ctx context.Context, name string) ([]model.Item, error) {
	ctx, span := a.tracer.Start(ctx, "list", trace.WithAttributes(attribute.String("name", name)))
	defer span.End()

	output, err := a.storage.List(ctx, name)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return output, err
}

func (a Service) WriteTo(ctx context.Context, name string, reader io.Reader, opts model.WriteOpts) error {
	ctx, span := a.tracer.Start(ctx, "writeTo", trace.WithAttributes(attribute.String("name", name)))
	defer span.End()

	err := a.storage.WriteTo(ctx, name, reader, opts)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (a Service) ReadFrom(ctx context.Context, name string) (model.ReadAtSeekCloser, error) {
	ctx, span := a.tracer.Start(ctx, "readFrom", trace.WithAttributes(attribute.String("name", name)))

	reader, err := a.storage.ReadFrom(ctx, name)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return telemetryCloser{
		ReadAtSeekCloser: reader,
		end:              span.End,
	}, err
}

func (a Service) UpdateDate(ctx context.Context, name string, date time.Time) error {
	ctx, span := a.tracer.Start(ctx, "updateDate", trace.WithAttributes(attribute.String("name", name)))
	defer span.End()

	err := a.storage.UpdateDate(ctx, name, date)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (a Service) Walk(ctx context.Context, name string, walkFn func(model.Item) error) error {
	ctx, span := a.tracer.Start(ctx, "walk", trace.WithAttributes(attribute.String("name", name)))
	defer span.End()

	err := a.storage.Walk(ctx, name, walkFn)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (a Service) Mkdir(ctx context.Context, name string, perm os.FileMode) error {
	ctx, span := a.tracer.Start(ctx, "mkdir", trace.WithAttributes(attribute.String("name", name)))
	defer span.End()

	err := a.storage.Mkdir(ctx, name, perm)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (a Service) Rename(ctx context.Context, oldName, newName string) error {
	ctx, span := a.tracer.Start(ctx, "rename", trace.WithAttributes(attribute.String("oldName", oldName)), trace.WithAttributes(attribute.String("newName", newName)))
	defer span.End()

	err := a.storage.Rename(ctx, oldName, newName)
	if err != nil {
		span.SetStatus(codes.Error, err.Error())
	}

	return err
}

func (a Service) RemoveAll(ctx context.Context, name string) error {
	ctx, span := a.tracer.Start(ctx, "removeAll", trace.WithAttributes(attribute.String("name", name)))
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

func (a Service) ConvertError(err error) error {
	return a.storage.ConvertError(err)
}
