package cms

type CreateBlogDto struct {
	Title          string `json:"title" binding:"required,max=200"`
	Slug           string `json:"slug" binding:"required,max=220"`
	Description    string `json:"description"`
	Content        string `json:"content" binding:"required"`
	SeoTitle       string `json:"seo_title" binding:"max=200"`
	SeoDescription string `json:"seo_description" binding:"max=500"`
	Keywords       string `json:"keywords" binding:"max=500"`
	MediaIDs       []uint `json:"media_ids"`
}

type UpdateBlogDto struct {
	Title          *string `json:"title" binding:"omitempty,max=200"`
	Slug           *string `json:"slug" binding:"omitempty,max=220"`
	Description    *string `json:"description"`
	Content        *string `json:"content"`
	SeoTitle       *string `json:"seo_title" binding:"omitempty,max=200"`
	SeoDescription *string `json:"seo_description" binding:"omitempty,max=500"`
	Keywords       *string `json:"keywords" binding:"omitempty,max=500"`
	MediaIDs       *[]uint `json:"media_ids"`
}

type CreateBannerDto struct {
	Title        string `json:"title" binding:"required,max=200"`
	MediaID      uint   `json:"media_id" binding:"required,min=1"`
	AltName      string `json:"alt_name" binding:"required,max=200"`
	LinkUrl      string `json:"link_url" binding:"omitempty,max=500"`
	DisplayOrder int    `json:"display_order"`
	IsActive     *bool  `json:"is_active"`
}

type UpdateBannerDto struct {
	Title        *string `json:"title" binding:"omitempty,max=200"`
	MediaID      *uint   `json:"media_id" binding:"omitempty,min=1"`
	AltName      *string `json:"alt_name" binding:"omitempty,max=200"`
	LinkUrl      *string `json:"link_url" binding:"omitempty,max=500"`
	DisplayOrder *int    `json:"display_order"`
	IsActive     *bool   `json:"is_active"`
}

type SiteContentItemDto struct {
	Key         string  `json:"key" binding:"required,max=100" example:"h1_title"`
	Value       string  `json:"value"`
	Description *string `json:"description"`
	MediaID     *uint   `json:"media_id" binding:"omitempty,min=1"`
}

type UpdateSiteContentDto struct {
	Items []SiteContentItemDto `json:"items" binding:"required,min=1,dive"`
}

type ContentResponse struct {
	Contents []SiteContent `json:"contents"`
	Banners  []Banner      `json:"banners"`
}

type BlogQuery struct {
	Page   int    `form:"page,default=1" binding:"min=1"`
	Limit  int    `form:"limit,default=20" binding:"min=1,max=100"`
	Search string `form:"search"`
}
