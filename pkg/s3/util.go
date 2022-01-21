package s3

import (
	"path"
	"strings"

	"github.com/ViBiOh/absto/pkg/model"
	"github.com/minio/minio-go/v7"
)

func convertToItem(info minio.ObjectInfo) model.Item {
	name := path.Base(info.Key)

	return model.Item{
		Name:      name,
		Pathname:  "/" + strings.TrimSuffix(info.Key, "/"),
		Extension: strings.ToLower(path.Ext(name)),
		IsDir:     strings.HasSuffix(info.Key, "/"),
		Date:      info.LastModified,
		Size:      info.Size,
	}
}

func convertError(err error) error {
	if err == nil {
		return err
	}

	if strings.Contains(err.Error(), "The specified key does not exist") {
		return model.ErrNotExist(err)
	}

	return err
}

// Dirname ensures given name is a dirname, with a trailing slash
func dirname(name string) string {
	if !strings.HasSuffix(name, "/") {
		return name + "/"
	}
	return name
}
