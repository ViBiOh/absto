package s3

import (
	"os"
	"path"
	"strings"

	"github.com/ViBiOh/absto/pkg/model"
	"github.com/minio/minio-go/v7"
)

func convertToItem(info minio.ObjectInfo) model.Item {
	name := path.Base(info.Key)
	pathname := "/" + info.Key

	item := model.Item{
		ID:         model.ID(pathname),
		NameValue:  name,
		Pathname:   pathname,
		IsDirValue: strings.HasSuffix(info.Key, "/"),
		Date:       info.LastModified,
	}

	if !item.IsDir() {
		item.Extension = strings.ToLower(path.Ext(name))
		item.SizeValue = info.Size
	} else {
		item.FileMode = uint32(os.ModeDir)
	}

	return item
}
