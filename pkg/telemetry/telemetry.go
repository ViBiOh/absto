package telemetry

import (
	"context"
	"io"
	"io/fs"
	"time"

	"github.com/ViBiOh/absto/pkg/model"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

var _ model.Storage = Service{}

type Service struct {
	model.Storage
	tracer trace.Tracer
}

func New(storage model.Storage, tracerProvider trace.TracerProvider) model.Storage {
	if tracerProvider == nil {
		return storage
	}

	return Service{
		Storage: storage,
		tracer:  tracerProvider.Tracer("absto"),
	}
}

func (a Service) Name() string {
	return a.Storage.Name()
}

func (a Service) WithIgnoreFn(ignoreFn func(model.Item) bool) model.Storage {
	return Service{
		Storage: a.Storage.WithIgnoreFn(ignoreFn),
		tracer:  a.tracer,
	}
}

func (a Service) Stat(ctx context.Context, name string) (model.Item, error) {
	ctx, span := a.tracer.Start(ctx, "stat", trace.WithAttributes(attribute.String("name", name)))
	defer span.End()

	output, err := a.Storage.Stat(ctx, name)
	recordError(span, err)

	return output, err
}

func (a Service) List(ctx context.Context, name string) ([]model.Item, error) {
	ctx, span := a.tracer.Start(ctx, "list", trace.WithAttributes(attribute.String("name", name)))
	defer span.End()

	output, err := a.Storage.List(ctx, name)
	recordError(span, err)

	return output, err
}

func (a Service) WriteTo(ctx context.Context, name string, reader io.Reader, opts model.WriteOpts) error {
	ctx, span := a.tracer.Start(ctx, "writeTo", trace.WithAttributes(attribute.String("name", name)))
	defer span.End()

	err := a.Storage.WriteTo(ctx, name, reader, opts)
	recordError(span, err)

	return err
}

func (a Service) ReadFrom(ctx context.Context, name string) (model.ReadAtSeekCloser, error) {
	ctx, span := a.tracer.Start(ctx, "readFrom", trace.WithAttributes(attribute.String("name", name)))

	reader, err := a.Storage.ReadFrom(ctx, name)
	if err != nil {
		recordError(span, err)
		span.End()

		return nil, err
	}

	closer := telemetryCloser{
		ReadAtSeekCloser: reader,
		span:             span,
	}

	if file, ok := reader.(statter); ok {
		return telemetryFile{telemetryCloser: closer, statter: file}, nil
	}

	return closer, nil
}

func (a Service) UpdateDate(ctx context.Context, name string, date time.Time) error {
	ctx, span := a.tracer.Start(ctx, "updateDate", trace.WithAttributes(attribute.String("name", name)))
	defer span.End()

	err := a.Storage.UpdateDate(ctx, name, date)
	recordError(span, err)

	return err
}

func (a Service) Walk(ctx context.Context, name string, walkFn func(model.Item) error) error {
	ctx, span := a.tracer.Start(ctx, "walk", trace.WithAttributes(attribute.String("name", name)))
	defer span.End()

	err := a.Storage.Walk(ctx, name, walkFn)
	recordError(span, err)

	return err
}

func (a Service) Mkdir(ctx context.Context, name string) error {
	ctx, span := a.tracer.Start(ctx, "mkdir", trace.WithAttributes(attribute.String("name", name)))
	defer span.End()

	err := a.Storage.Mkdir(ctx, name)
	recordError(span, err)

	return err
}

func (a Service) Rename(ctx context.Context, oldName, newName string) error {
	ctx, span := a.tracer.Start(ctx, "rename", trace.WithAttributes(attribute.String("oldName", oldName)), trace.WithAttributes(attribute.String("newName", newName)))
	defer span.End()

	err := a.Storage.Rename(ctx, oldName, newName)
	recordError(span, err)

	return err
}

func (a Service) RemoveAll(ctx context.Context, name string) error {
	ctx, span := a.tracer.Start(ctx, "removeAll", trace.WithAttributes(attribute.String("name", name)))
	defer span.End()

	err := a.Storage.RemoveAll(ctx, name)
	recordError(span, err)

	return err
}

type statter interface {
	Stat() (fs.FileInfo, error)
}

type telemetryCloser struct {
	model.ReadAtSeekCloser
	span trace.Span
}

func (closer telemetryCloser) Close() error {
	err := closer.ReadAtSeekCloser.Close()

	recordError(closer.span, err)
	closer.span.End()

	return err
}

type telemetryFile struct {
	statter
	telemetryCloser
}

func recordError(span trace.Span, err error) {
	if err == nil || model.IsNotExist(err) {
		return
	}

	span.RecordError(err)
	span.SetStatus(codes.Error, err.Error())
}

func (a Service) ConvertError(err error) error {
	return a.Storage.ConvertError(err)
}
