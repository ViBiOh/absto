package s3

import (
	"path"
	"strings"

	"github.com/ViBiOh/absto/pkg/model"
	"github.com/minio/minio-go/v7"
)

func convertToItem(info minio.ObjectInfo) model.Item {
	return model.Item{
		Name:     path.Base(info.Key),
		Pathname: info.Key,
		IsDir:    strings.HasSuffix(info.Key, "/"),
		Date:     info.LastModified,
		Size:     info.Size,
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
