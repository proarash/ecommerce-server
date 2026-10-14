package product

type CreateCategoryDto struct {
	Name           string   `json:"name" binding:"required,max=100"`
	SeoTitle       string   `json:"seo_title" binding:"max=200"`
	SeoDescription string   `json:"seo_description" binding:"max=500"`
	SeoKeywords    []string `json:"seo_keywords" binding:"max=20,dive,max=100"`
	Slug           string   `json:"slug" binding:"required,max=120"`
	ParentID       *uint    `json:"parent_id" binding:"omitempty,min=1"`
}

type UpdateCategoryDto struct {
	Name           *string   `json:"name" binding:"omitempty,max=100"`
	SeoTitle       *string   `json:"seo_title" binding:"omitempty,max=200"`
	SeoDescription *string   `json:"seo_description" binding:"omitempty,max=500"`
	SeoKeywords    *[]string `json:"seo_keywords" binding:"omitempty,max=20,dive,max=100"`
	Slug           *string   `json:"slug" binding:"omitempty,max=120"`
	ParentID       *uint     `json:"parent_id" binding:"omitempty,min=1,excluded_with=MakeRoot"`
	MakeRoot       bool      `json:"make_root"`
}

type CreateProductDto struct {
	Title          string                 `json:"title" binding:"required,max=200" example:"پاوربانک اوریکو K20"`
	Slug           string                 `json:"slug" binding:"max=220"`
	Description    string                 `json:"description" example:"پاوربانک ۲۰۰۰۰ میلی‌آمپر با شارژ سریع"`
	SeoTitle       string                 `json:"seo_title" binding:"max=200" example:"خرید پاوربانک اوریکو K20 | فروشگاه"`
	SeoDescription string                 `json:"seo_description" binding:"max=500"`
	SeoKeywords    []string               `json:"seo_keywords" binding:"max=20,dive,max=100"`
	Price          float64                `json:"price" binding:"required,gt=0" example:"3900000"`
	SKU            string                 `json:"sku" binding:"required,max=64" example:"POWER-BANK-001"`
	IsActive       *bool                  `json:"is_active"`
	CategoryID     uint                   `json:"category_id" binding:"required,min=1" example:"13"`
	Attributes     []AttributeValuesInput `json:"attributes" binding:"omitempty,max=100,dive"`
}

type UpdateProductDto struct {
	Title          *string                 `json:"title" binding:"omitempty,max=200"`
	Slug           *string                 `json:"slug" binding:"omitempty,max=220"`
	Description    *string                 `json:"description"`
	SeoTitle       *string                 `json:"seo_title" binding:"omitempty,max=200"`
	SeoDescription *string                 `json:"seo_description" binding:"omitempty,max=500"`
	SeoKeywords    *[]string               `json:"seo_keywords" binding:"omitempty,max=20,dive,max=100"`
	Price          *float64                `json:"price" binding:"omitempty,gt=0"`
	SKU            *string                 `json:"sku" binding:"omitempty,max=64"`
	IsActive       *bool                   `json:"is_active"`
	CategoryID     *uint                   `json:"category_id" binding:"omitempty,min=1"`
	Attributes     *[]AttributeValuesInput `json:"attributes" binding:"omitempty,max=100,dive"`
}

type ProductQuery struct {
	Page       int    `form:"page,default=1" binding:"min=1"`
	Limit      int    `form:"limit,default=20" binding:"min=1,max=100"`
	CategoryID uint   `form:"category_id"`
	Search     string `form:"search"`
}

type CategoryTree struct {
	ID             uint           `json:"id"`
	Name           string         `json:"name"`
	Slug           string         `json:"slug"`
	SeoTitle       string         `json:"seo_title"`
	SeoDescription string         `json:"seo_description"`
	SeoKeywords    []string       `json:"seo_keywords"`
	ParentID       *uint          `json:"parent_id"`
	MediaID        *uint          `json:"media_id"`
	MediaURL       *string        `json:"media_url"`
	Children       []CategoryTree `json:"children"`
}

type AttributeValuesInput struct {
	AttributeID uint     `json:"attribute_id" binding:"required,min=1" example:"1"`
	Values      []string `json:"values" binding:"required,min=1,max=50,dive,required,max=500" example:"قرمز,آبی"`
}

type CreateAttributeDto struct {
	Key      string `json:"key" binding:"required,max=100" example:"مدل"`
	Name     string `json:"name" binding:"required,max=200" example:"مدل"`
	DataType string `json:"data_type" binding:"omitempty,oneof=text number boolean" enums:"text,number,boolean" example:"text"`
}

type UpdateAttributeDto struct {
	Key      *string `json:"key" binding:"omitempty,max=100"`
	Name     *string `json:"name" binding:"omitempty,max=200"`
	DataType *string `json:"data_type" binding:"omitempty,oneof=text number boolean" enums:"text,number,boolean" example:"text"`
}

type AttributeQuery struct {
	Page   int    `form:"page,default=1" binding:"min=1"`
	Limit  int    `form:"limit,default=20" binding:"min=1,max=100"`
	Search string `form:"search"`
}

type AssignAttributesDto struct {
	Attributes []AttributeValuesInput `json:"attributes" binding:"required,min=1,max=100,dive"`
}

type DetachAttributeQuery struct {
	Value string `form:"value"`
}

type AssignMediaDto struct {
	MediaIDs []uint `json:"media_ids" binding:"required,min=1,dive,min=1"`
}
