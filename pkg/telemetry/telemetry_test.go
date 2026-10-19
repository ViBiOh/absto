package telemetry

import (
	"context"
	"io"
	"io/fs"
	"os"
	"path/filepath"
	"testing"

	"github.com/ViBiOh/absto/pkg/filesystem"
	"github.com/ViBiOh/absto/pkg/model"
	"go.opentelemetry.io/otel/codes"
	sdktrace "go.opentelemetry.io/otel/sdk/trace"
	"go.opentelemetry.io/otel/sdk/trace/tracetest"
)

func newTestStorage(t *testing.T) (model.Storage, *tracetest.SpanRecorder) {
	t.Helper()

	root := t.TempDir()
	if err := os.WriteFile(filepath.Join(root, "file.txt"), []byte("content"), 0o600); err != nil {
		t.Fatal(err)
	}

	backend, err := filesystem.New(root)
	if err != nil {
		t.Fatal(err)
	}

	recorder := tracetest.NewSpanRecorder()
	provider := sdktrace.NewTracerProvider(sdktrace.WithSpanProcessor(recorder))

	t.Cleanup(func() { _ = provider.Shutdown(context.Background()) })

	return New(backend, provider), recorder
}

func TestNewWithoutProvider(t *testing.T) {
	t.Parallel()

	backend, err := filesystem.New(t.TempDir())
	if err != nil {
		t.Fatal(err)
	}

	if got := New(backend, nil); got.Name() != filesystem.Name {
		t.Errorf("New() = %T, want the backend itself", got)
	}

	if _, wrapped := New(backend, nil).(Service); wrapped {
		t.Error("New() wrapped the backend without a tracer provider")
	}
}

func TestSpanStatus(t *testing.T) {
	t.Parallel()

	cases := map[string]struct {
		name       string
		wantStatus codes.Code
		wantEvents int
	}{
		"success": {
			"/file.txt",
			codes.Unset,
			0,
		},
		"not exist": {
			"/missing.txt",
			codes.Unset,
			0,
		},
		"error": {
			"/../file.txt",
			codes.Error,
			1,
		},
	}

	for intention, tc := range cases {
		t.Run(intention, func(t *testing.T) {
			t.Parallel()

			storage, recorder := newTestStorage(t)

			_, _ = storage.Stat(context.Background(), tc.name)

			spans := recorder.Ended()
			if len(spans) != 1 {
				t.Fatalf("got %d ended spans, want 1", len(spans))
			}

			if got := spans[0].Status().Code; got != tc.wantStatus {
				t.Errorf("status = %s, want %s", got, tc.wantStatus)
			}

			if got := len(spans[0].Events()); got != tc.wantEvents {
				t.Errorf("events = %d, want %d", got, tc.wantEvents)
			}
		})
	}
}

func TestReadFrom(t *testing.T) {
	t.Parallel()

	t.Run("span covers close", func(t *testing.T) {
		t.Parallel()

		storage, recorder := newTestStorage(t)

		reader, err := storage.ReadFrom(context.Background(), "/file.txt")
		if err != nil {
			t.Fatal(err)
		}

		if got := len(recorder.Ended()); got != 0 {
			t.Fatalf("got %d ended spans before Close, want 0", got)
		}

		content, err := io.ReadAll(reader)
		if err != nil {
			t.Fatal(err)
		}

		if string(content) != "content" {
			t.Errorf("content = %q, want `content`", content)
		}

		if err = reader.Close(); err != nil {
			t.Fatal(err)
		}

		if got := len(recorder.Ended()); got != 1 {
			t.Errorf("got %d ended spans after Close, want 1", got)
		}
	})

	t.Run("not exist", func(t *testing.T) {
		t.Parallel()

		storage, recorder := newTestStorage(t)

		if _, err := storage.ReadFrom(context.Background(), "/missing.txt"); !model.IsNotExist(err) {
			t.Fatalf("ReadFrom() error = %v, want not exist", err)
		}

		spans := recorder.Ended()
		if len(spans) != 1 {
			t.Fatalf("got %d ended spans, want 1", len(spans))
		}

		if got := spans[0].Status().Code; got != codes.Unset {
			t.Errorf("status = %s, want %s", got, codes.Unset)
		}
	})

	t.Run("stat is preserved", func(t *testing.T) {
		t.Parallel()

		storage, _ := newTestStorage(t)

		reader, err := storage.ReadFrom(context.Background(), "/file.txt")
		if err != nil {
			t.Fatal(err)
		}

		defer func() { _ = reader.Close() }()

		file, ok := reader.(interface{ Stat() (fs.FileInfo, error) })
		if !ok {
			t.Fatalf("%T does not expose Stat()", reader)
		}

		info, err := file.Stat()
		if err != nil {
			t.Fatal(err)
		}

		if info.Size() != int64(len("content")) {
			t.Errorf("Stat().Size() = %d, want %d", info.Size(), len("content"))
		}
	})
}
