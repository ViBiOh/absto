package filesystem

import (
	"testing"

	"github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/absto/pkg/storagetest"
)

func TestStorageContract(t *testing.T) {
	t.Parallel()

	storagetest.Run(t, func(t *testing.T, _ string) model.Storage {
		t.Helper()

		service, err := New(t.TempDir())
		if err != nil {
			t.Fatal(err)
		}

		return service
	})
}
