package model

import (
	"io/fs"
	"os"
	"time"
)

var _ fs.FileInfo = ItemInfo{}

type ItemInfo struct {
	Item
}

func (ii ItemInfo) Name() string {
	return ii.Item.Name
}

func (ii ItemInfo) Size() int64 {
	return ii.Item.Size
}

func (ii ItemInfo) Mode() os.FileMode {
	return ii.Item.FileMode
}

func (ii ItemInfo) ModTime() time.Time {
	return ii.Item.Date
}

func (ii ItemInfo) IsDir() bool {
	return ii.Item.IsDir
}

func (ii ItemInfo) Sys() any {
	return nil
}
