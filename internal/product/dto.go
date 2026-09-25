package product

type CreateCategoryDto struct {
	Name     string `json:"name" binding:"required,max=100"`
	Slug     string `json:"slug" binding:"required,max=120"`
	ParentID *uint  `json:"parent_id" binding:"omitempty,min=1"`
	MediaID  *uint  `json:"media_id" binding:"omitempty,min=1"`
}

type UpdateCategoryDto struct {
	Name     *string `json:"name" binding:"omitempty,max=100"`
	Slug     *string `json:"slug" binding:"omitempty,max=120"`
	ParentID *uint   `json:"parent_id" binding:"omitempty,min=1"`
	MediaID  *uint   `json:"media_id" binding:"omitempty,min=1"`
}

type CreateProductDto struct {
	Title        string  `json:"title" binding:"required,max=200"`
	Description  string  `json:"description"`
	Price        float64 `json:"price" binding:"required,gt=0"`
	SKU          string  `json:"sku" binding:"required,max=64"`
	IsActive     *bool   `json:"is_active"`
	CategoryID   uint    `json:"category_id" binding:"required,min=1"`
	MediaIDs     []uint  `json:"media_ids"`
	AttributeIDs []uint  `json:"attribute_ids"`
}

type UpdateProductDto struct {
	Title        *string  `json:"title" binding:"omitempty,max=200"`
	Description  *string  `json:"description"`
	Price        *float64 `json:"price" binding:"omitempty,gt=0"`
	SKU          *string  `json:"sku" binding:"omitempty,max=64"`
	IsActive     *bool    `json:"is_active"`
	CategoryID   *uint    `json:"category_id" binding:"omitempty,min=1"`
	MediaIDs     *[]uint  `json:"media_ids"`
	AttributeIDs *[]uint  `json:"attribute_ids"`
}

type ProductQuery struct {
	Page       int    `form:"page,default=1" binding:"min=1"`
	Limit      int    `form:"limit,default=20" binding:"min=1,max=100"`
	CategoryID uint   `form:"category_id"`
	Search     string `form:"search"`
}

type CategoryTree struct {
	ID       uint           `json:"id"`
	Name     string         `json:"name"`
	Slug     string         `json:"slug"`
	ParentID *uint          `json:"parent_id"`
	MediaID  *uint          `json:"media_id"`
	MediaURL *string        `json:"media_url"`
	Children []CategoryTree `json:"children"`
}

type CreateAttributeDto struct {
	Key   string `json:"key" binding:"required,max=100" example:"color"`
	Title string `json:"title" binding:"required,max=200" example:"Color"`
	Name  string `json:"name" binding:"required,max=200" example:"Red"`
}

type UpdateAttributeDto struct {
	Key   *string `json:"key" binding:"omitempty,max=100"`
	Title *string `json:"title" binding:"omitempty,max=200"`
	Name  *string `json:"name" binding:"omitempty,max=200"`
}

type AttributeQuery struct {
	Page   int    `form:"page,default=1" binding:"min=1"`
	Limit  int    `form:"limit,default=20" binding:"min=1,max=100"`
	Search string `form:"search"`
}

type AssignAttributesDto struct {
	AttributeIDs []uint `json:"attribute_ids" binding:"required,min=1,dive,min=1"`
}
