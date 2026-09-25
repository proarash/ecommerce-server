package product

import (
	"github.com/proarash/ecommerce-server/internal/media"
	"gorm.io/gorm"
)

type Category struct {
	gorm.Model
	Name     string       `json:"name"`
	Slug     string       `json:"slug" gorm:"uniqueIndex;not null"`
	ParentID *uint        `json:"parent_id" gorm:"index"`
	Parent   *Category    `json:"parent,omitempty" swaggerignore:"true"`
	Children []Category   `json:"children,omitempty" gorm:"foreignKey:ParentID" swaggerignore:"true"`
	MediaID  *uint        `json:"media_id"`
	Media    *media.Media `json:"media,omitempty"`
	Products []Product    `json:"products,omitempty" swaggerignore:"true"`
}

type Product struct {
	gorm.Model
	Title       string        `json:"title" gorm:"uniqueIndex;not null"`
	Description string        `json:"description"`
	Price       float64       `json:"price"`
	SKU         string        `json:"sku" gorm:"uniqueIndex;not null"`
	IsActive    bool          `json:"is_active" gorm:"default:true"`
	CategoryID  uint          `json:"category_id" gorm:"index;not null"`
	Category    *Category     `json:"category,omitempty"`
	Media       []media.Media `json:"media,omitempty" gorm:"many2many:product_media"`
	Attributes  []Attribute   `json:"attributes,omitempty" gorm:"many2many:product_attributes"`
}

type Attribute struct {
	gorm.Model
	Key      string    `json:"key" gorm:"uniqueIndex;not null"`
	Title    string    `json:"title"`
	Name     string    `json:"name"`
	Products []Product `json:"products,omitempty" gorm:"many2many:product_attributes" swaggerignore:"true"`
}
