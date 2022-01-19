package model

import (
	"path"
	"path/filepath"
	"strings"
	"time"
)

var (
	// ImageExtensions contains extensions of Image
	ImageExtensions = map[string]bool{".jpg": true, ".jpeg": true, ".png": true, ".gif": true, ".svg": true, ".tiff": true}
	// PdfExtensions contains extensions of Pdf
	PdfExtensions = map[string]bool{".pdf": true}
	// VideoExtensions contains extensions of Video
	VideoExtensions = map[string]bool{".mp4": true, ".mov": true, ".avi": true}
)

// Item describe item on a storage provider
type Item struct {
	Date     time.Time `json:"date"`
	Name     string    `json:"name"`
	Pathname string    `json:"pathname"`
	IsDir    bool      `json:"isDir"`
	Size     int64     `json:"size"`
}

// Extension gives extensions of item
func (s Item) Extension() string {
	return strings.ToLower(path.Ext(s.Name))
}

// IsPdf determine if item if a pdf
func (s Item) IsPdf() bool {
	return PdfExtensions[s.Extension()]
}

// IsImage determine if item if an image
func (s Item) IsImage() bool {
	return ImageExtensions[s.Extension()]
}

// IsVideo determine if item if a video
func (s Item) IsVideo() bool {
	return VideoExtensions[s.Extension()]
}

// Dir return the nearest directory (self of parent)
func (s Item) Dir() string {
	if s.IsDir {
		return s.Pathname
	}

	return filepath.Dir(s.Pathname)
}

// Dirname ensures given name is a dirname, with a trailing slash
func Dirname(name string) string {
	if !strings.HasSuffix(name, "/") {
		return name + "/"
	}
	return name
}
