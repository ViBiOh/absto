package s3

import (
	"os"
	"testing"

	"github.com/ViBiOh/absto/pkg/model"
	"github.com/ViBiOh/absto/pkg/storagetest"
)

func TestStorageContract(t *testing.T) {
	t.Parallel()

	if os.Getenv("ABSTO_TEST_ENABLED") != "true" {
		return
	}

	storagetest.Run(t, func(t *testing.T, prefix string) model.Storage {
		t.Helper()

		Endpoint := os.Getenv("ABSTO_TEST_ENDPOINT")
		AccessKey := os.Getenv("ABSTO_TEST_ACCESS_KEY")
		SecretAccess := os.Getenv("ABSTO_TEST_SECRET_ACCESS")
		Bucket := os.Getenv("ABSTO_TEST_BUCKET")

		service, err := New(Endpoint, AccessKey, SecretAccess, Bucket, true, 5<<20, WithPrefix(prefix))
		if err != nil {
			t.Fatal(err)
		}

		return service
	})
}
