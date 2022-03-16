package s3

import (
	"path"
	"strings"

	"github.com/ViBiOh/absto/pkg/model"
	"github.com/minio/minio-go/v7"
)

func convertToItem(info minio.ObjectInfo) model.Item {
	name := path.Base(info.Key)
	pathname := "/" + info.Key

	return model.Item{
		ID:        model.Sha(pathname),
		Name:      name,
		Pathname:  pathname,
		Extension: strings.ToLower(path.Ext(name)),
		IsDir:     strings.HasSuffix(info.Key, "/"),
		Date:      info.LastModified,
		Size:      info.Size,
	}
}

// Dirname ensures given name is a dirname, with a trailing slash
func dirname(name string) string {
	if !strings.HasSuffix(name, "/") {
		return name + "/"
	}
	return name
}
