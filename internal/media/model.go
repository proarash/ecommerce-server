package media

import "time"

const TypeImage = "image"

type Media struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	URL       string    `json:"url"`
	MediaType string    `json:"media_type" gorm:"index"`
	MimeType  string    `json:"mime_type"`
	FileSize  int64     `json:"file_size"`
	FileName  string    `json:"file_name" gorm:"uniqueIndex"`
	Title     string    `json:"title" gorm:"not null;default:''"`
	Alt       string    `json:"alt" gorm:"not null;default:''"`
}
