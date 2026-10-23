package s3

import (
	"errors"
	"io/fs"
	"path"
	"strings"

	"github.com/ViBiOh/absto/pkg/model"
	"github.com/minio/minio-go/v7"
)

const storageClassHeader = "X-Amz-Storage-Class"

var errStopWalk = errors.New("stop walk")

func (s Service) ConvertError(err error) error {
	if err == nil {
		return nil
	}

	if IsNotExist(err) {
		return model.ErrNotExist(err)
	}

	return err
}

func IsNotExist(err error) bool {
	if err == nil {
		return false
	}

	var response minio.ErrorResponse

	return errors.As(err, &response) && response.Code == minio.NoSuchKey
}

func (s Service) copyDestination(object string) minio.CopyDestOptions {
	options := minio.CopyDestOptions{
		Bucket: s.bucket,
		Object: object,
	}

	if len(s.storageClass) != 0 {
		options.ReplaceMetadata = true
		options.UserMetadata = map[string]string{storageClassHeader: s.storageClass}
	}

	return options
}

func notExist() error {
	return model.ErrNotExist(fs.ErrNotExist)
}

func (s Service) relative(name string) string {
	if len(name) != 0 && name[0] == '/' {
		name = name[1:]
	}

	if len(s.prefix) == 0 {
		return name
	}

	return s.prefix + "/" + name
}

func (s Service) isRoot(key string) bool {
	return strings.Trim(key, "/") == s.prefix
}

func dirKey(key string) string {
	if len(key) == 0 {
		return key
	}

	return model.Dirname(key)
}

func (s Service) convertToItem(info minio.ObjectInfo) model.Item {
	relative := strings.TrimPrefix(info.Key, s.prefix)
	name := path.Base(relative)
	pathname := path.Clean("/" + strings.TrimSuffix(relative, "/"))

	item := model.Item{
		ID:         model.ID(pathname),
		NameValue:  name,
		Pathname:   pathname,
		IsDirValue: strings.HasSuffix(relative, "/"),
		Date:       info.LastModified,
	}

	if !item.IsDir() {
		item.Extension = strings.ToLower(path.Ext(name))
		item.SizeValue = info.Size
		item.FileMode = model.RegularFilePerm
	} else {
		item.FileMode = model.DirectoryPerm
	}

	return item
}

func rootItem() model.Item {
	return model.Item{
		ID:         model.ID("/"),
		NameValue:  "/",
		Pathname:   "/",
		IsDirValue: true,
		FileMode:   model.DirectoryPerm,
	}
}
