package s3

import (
	"testing"

	"github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/absto/pkg/storagetest"
)

func TestStorageContract(t *testing.T) {
	t.Parallel()

	storagetest.Run(t, func(t *testing.T) model.Storage {
		t.Helper()

		service, _ := newTestService(t, false)

		return service
	})
}
