package product

import (
	"sort"
	"time"

	"github.com/proarash/ecommerce-server/internal/media"
	"gorm.io/gorm"
)

type Category struct {
	gorm.Model
	Name           string       `json:"name"`
	SeoTitle       string       `json:"seo_title"`
	SeoDescription string       `json:"seo_description"`
	SeoKeywords    Keywords     `json:"seo_keywords" gorm:"type:jsonb;not null;default:'[]'" swaggertype:"array,string"`
	Slug           string       `json:"slug" gorm:"uniqueIndex;not null"`
	ParentID       *uint        `json:"parent_id" gorm:"index"`
	Parent         *Category    `json:"parent,omitempty" swaggerignore:"true"`
	Children       []Category   `json:"children,omitempty" gorm:"foreignKey:ParentID" swaggerignore:"true"`
	MediaID        *uint        `json:"media_id"`
	Media          *media.Media `json:"media,omitempty"`
	Products       []Product    `json:"products,omitempty" swaggerignore:"true"`
}

type Product struct {
	gorm.Model
	Title           string                  `json:"title" gorm:"uniqueIndex;not null"`
	Slug            string                  `json:"slug" gorm:"uniqueIndex;not null"`
	Description     string                  `json:"description"`
	SeoTitle        string                  `json:"seo_title"`
	SeoDescription  string                  `json:"seo_description"`
	SeoKeywords     Keywords                `json:"seo_keywords" gorm:"type:jsonb;not null;default:'[]'" swaggertype:"array,string"`
	Price           float64                 `json:"price"`
	SKU             string                  `json:"sku" gorm:"uniqueIndex;not null"`
	IsActive        bool                    `json:"is_active" gorm:"default:true"`
	CategoryID      uint                    `json:"category_id" gorm:"index;not null"`
	Category        *Category               `json:"category,omitempty"`
	Media           []media.Media           `json:"media,omitempty" gorm:"many2many:product_media"`
	AttributeValues []ProductAttributeValue `json:"-" gorm:"constraint:OnDelete:CASCADE"`
	Attributes      []ProductAttribute      `json:"attributes" gorm:"-"`
}

func (p *Product) AfterFind(*gorm.DB) error {
	p.Attributes = groupAttributeValues(p.AttributeValues)
	return nil
}

const (
	DataTypeText    = "text"
	DataTypeNumber  = "number"
	DataTypeBoolean = "boolean"
)

type Attribute struct {
	ID        uint      `json:"id" gorm:"primarykey"`
	CreatedAt time.Time `json:"created_at"`
	UpdatedAt time.Time `json:"updated_at"`
	Key       string    `json:"key" gorm:"uniqueIndex;not null" example:"مدل"`
	Name      string    `json:"name" gorm:"not null" example:"مدل"`
	DataType  string    `json:"data_type" gorm:"not null;default:'text'" enums:"text,number,boolean" example:"text"`
}

type ProductAttributeValue struct {
	ID          uint       `json:"id" gorm:"primarykey"`
	CreatedAt   time.Time  `json:"created_at"`
	UpdatedAt   time.Time  `json:"updated_at"`
	ProductID   uint       `json:"product_id" gorm:"not null;uniqueIndex:idx_product_attribute_value,priority:1;index:idx_product_attribute,priority:1"`
	AttributeID uint       `json:"attribute_id" gorm:"not null;uniqueIndex:idx_product_attribute_value,priority:2;index:idx_product_attribute,priority:2"`
	Attribute   *Attribute `json:"attribute,omitempty" gorm:"constraint:OnDelete:CASCADE"`
	Value       string     `json:"value" gorm:"type:text;not null;uniqueIndex:idx_product_attribute_value,priority:3" example:"آیفون ۱۶ پرو"`
}

type ProductAttribute struct {
	AttributeID uint     `json:"attribute_id" example:"1"`
	Key         string   `json:"key" example:"مدل"`
	Name        string   `json:"name" example:"مدل"`
	DataType    string   `json:"data_type" example:"text"`
	Values      []string `json:"values" example:"آیفون ۱۶ پرو,آیفون ۱۶"`
}

func groupAttributeValues(values []ProductAttributeValue) []ProductAttribute {
	sorted := append([]ProductAttributeValue(nil), values...)
	sort.Slice(sorted, func(i, j int) bool { return sorted[i].ID < sorted[j].ID })
	groups := []ProductAttribute{}
	index := map[uint]int{}
	for _, v := range sorted {
		i, ok := index[v.AttributeID]
		if !ok {
			g := ProductAttribute{AttributeID: v.AttributeID, Values: []string{}}
			if v.Attribute != nil {
				g.Key, g.Name, g.DataType = v.Attribute.Key, v.Attribute.Name, v.Attribute.DataType
			}
			i = len(groups)
			index[v.AttributeID] = i
			groups = append(groups, g)
		}
		groups[i].Values = append(groups[i].Values, v.Value)
	}
	return groups
}
