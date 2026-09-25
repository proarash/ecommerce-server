package media

import "gorm.io/gorm"

const (
	TypeImage    = "image"
	TypeVideo    = "video"
	TypeDocument = "document"
)

type Media struct {
	gorm.Model
	URL       string `json:"url"`
	MediaType string `json:"media_type" gorm:"index"`
	MimeType  string `json:"mime_type"`
	FileSize  int64  `json:"file_size"`
	FileName  string `json:"file_name"`
}
