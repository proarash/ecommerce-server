package cms

import (
	"github.com/proarash/ecommerce-server/internal/media"
	"gorm.io/gorm"
)

type BlogPost struct {
	gorm.Model
	Title          string        `json:"title"`
	Slug           string        `json:"slug" gorm:"uniqueIndex;not null"`
	Description    string        `json:"description"`
	Content        string        `json:"content"`
	SeoTitle       string        `json:"seo_title"`
	SeoDescription string        `json:"seo_description"`
	Keywords       string        `json:"keywords"`
	AuthorID       uint          `json:"author_id"`
	Media          []media.Media `json:"media,omitempty" gorm:"many2many:blog_media"`
}

type Banner struct {
	gorm.Model
	Title        string       `json:"title"`
	MediaID      uint         `json:"media_id"`
	Media        *media.Media `json:"media,omitempty"`
	AltName      string       `json:"alt_name"`
	LinkUrl      string       `json:"link_url"`
	DisplayOrder int          `json:"display_order" gorm:"default:0"`
	IsActive     bool         `json:"is_active" gorm:"default:true"`
}

type SiteContent struct {
	gorm.Model
	Key         string       `json:"key" gorm:"uniqueIndex;not null"`
	Value       string       `json:"value"`
	Description string       `json:"description"`
	MediaID     *uint        `json:"media_id"`
	Media       *media.Media `json:"media,omitempty"`
}
