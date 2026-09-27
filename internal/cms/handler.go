package cms

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/middleware"
	"github.com/proarash/ecommerce-server/internal/types"
)

type Handler struct {
	store Store
}

func NewHandler(store Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) RegisterRoutes(public gin.IRouter, marketer gin.IRouter) {
	public.GET("/cms/content", h.GetContent)
	public.GET("/cms/blogs", h.ListBlogs)
	public.GET("/cms/blogs/:slug", h.GetBlog)

	marketer.POST("/cms/blogs", h.CreateBlog)
	marketer.PATCH("/cms/blogs/:slug", h.UpdateBlog)
	marketer.DELETE("/cms/blogs/:slug", h.DeleteBlog)
	marketer.POST("/cms/banners", h.CreateBanner)
	marketer.PATCH("/cms/banners/:id", h.UpdateBanner)
	marketer.DELETE("/cms/banners/:id", h.DeleteBanner)
	marketer.PATCH("/cms/content", h.UpdateContent)
}

// GetContent godoc
// @Summary Static site settings and active banners
// @Tags CMS
// @Produce json
// @Success 200 {object} types.ApiResponse{data=ContentResponse}
// @Router /cms/content [get]
func (h *Handler) GetContent(c *gin.Context) {
	contents, err := h.store.ListContents(c.Request.Context())
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	banners, err := h.store.ActiveBanners(c.Request.Context())
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	c.JSON(http.StatusOK, ContentResponse{Contents: contents, Banners: banners})
}

// UpdateContent godoc
// @Summary Update site content
// @Description Upserts site content entries by key (H1 titles, footer links, descriptions, logos)
// @Tags CMS
// @Accept json
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param body body UpdateSiteContentDto true "Content items"
// @Success 200 {object} types.ApiResponse{data=ContentResponse}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /cms/content [patch]
func (h *Handler) UpdateContent(c *gin.Context) {
	var dto UpdateSiteContentDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	if err := h.store.UpsertContents(c.Request.Context(), dto.Items); err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	h.GetContent(c)
}

// ListBlogs godoc
// @Summary List blog posts
// @Tags CMS
// @Produce json
// @Param search query string false "Title search"
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} types.ApiResponse{data=types.PaginatedResponse{items=[]BlogPost}}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /cms/blogs [get]
func (h *Handler) ListBlogs(c *gin.Context) {
	var q BlogQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	items, total, err := h.store.ListBlogs(c.Request.Context(), q)
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	c.JSON(http.StatusOK, types.PaginatedResponse{Items: items, Total: total, Page: q.Page, Limit: q.Limit})
}

// GetBlog godoc
// @Summary Get blog post by slug
// @Tags CMS
// @Produce json
// @Param slug path string true "Blog slug"
// @Success 200 {object} types.ApiResponse{data=BlogPost}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /cms/blogs/{slug} [get]
func (h *Handler) GetBlog(c *gin.Context) {
	b, err := h.store.FindBlogBySlug(c.Request.Context(), c.Param("slug"))
	if err != nil {
		types.HandleError(c, err, "blog not found")
		return
	}
	c.JSON(http.StatusOK, b)
}

// CreateBlog godoc
// @Summary Create blog post
// @Tags CMS
// @Accept json
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param body body CreateBlogDto true "Blog post"
// @Success 201 {object} types.ApiResponse{data=BlogPost}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 409 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /cms/blogs [post]
func (h *Handler) CreateBlog(c *gin.Context) {
	var dto CreateBlogDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	b := BlogPost{Title: dto.Title, Slug: dto.Slug, Description: dto.Description, Content: dto.Content, SeoTitle: dto.SeoTitle, SeoDescription: dto.SeoDescription, Keywords: dto.Keywords, AuthorID: middleware.GetAuth(c).UserID}
	if err := h.store.CreateBlog(c.Request.Context(), &b, dto.MediaIDs); err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	c.JSON(http.StatusCreated, b)
}

// UpdateBlog godoc
// @Summary Update blog post
// @Description The path segment is the blog post ID. media_ids, when provided, replaces the attached media.
// @Tags CMS
// @Accept json
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param id path int true "Blog ID"
// @Param body body UpdateBlogDto true "Fields"
// @Success 200 {object} types.ApiResponse{data=BlogPost}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /cms/blogs/{id} [patch]
func (h *Handler) UpdateBlog(c *gin.Context) {
	id, ok := types.ParamID(c, "slug")
	if !ok {
		return
	}
	var dto UpdateBlogDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	fields := map[string]any{}
	set := func(k string, v *string) {
		if v != nil {
			fields[k] = *v
		}
	}
	set("title", dto.Title)
	set("slug", dto.Slug)
	set("description", dto.Description)
	set("content", dto.Content)
	set("seo_title", dto.SeoTitle)
	set("seo_description", dto.SeoDescription)
	set("keywords", dto.Keywords)
	if err := h.store.UpdateBlog(c.Request.Context(), id, fields, dto.MediaIDs); err != nil {
		types.HandleError(c, err, "blog not found")
		return
	}
	b, err := h.store.FindBlog(c.Request.Context(), id)
	if err != nil {
		types.HandleError(c, err, "blog not found")
		return
	}
	c.JSON(http.StatusOK, b)
}

// DeleteBlog godoc
// @Summary Delete blog post
// @Description The path segment is the blog post ID
// @Tags CMS
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param id path int true "Blog ID"
// @Success 200 {object} types.ApiResponse{data=types.MessageResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /cms/blogs/{id} [delete]
func (h *Handler) DeleteBlog(c *gin.Context) {
	id, ok := types.ParamID(c, "slug")
	if !ok {
		return
	}
	if err := h.store.DeleteBlog(c.Request.Context(), id); err != nil {
		types.HandleError(c, err, "blog not found")
		return
	}
	c.JSON(http.StatusOK, types.MessageResponse{Message: "deleted"})
}

// CreateBanner godoc
// @Summary Create banner
// @Tags CMS
// @Accept json
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param body body CreateBannerDto true "Banner"
// @Success 201 {object} types.ApiResponse{data=Banner}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /cms/banners [post]
func (h *Handler) CreateBanner(c *gin.Context) {
	var dto CreateBannerDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	b := Banner{Title: dto.Title, MediaID: dto.MediaID, AltName: dto.AltName, LinkUrl: dto.LinkUrl, DisplayOrder: dto.DisplayOrder, IsActive: true}
	if err := h.store.CreateBanner(c.Request.Context(), &b); err != nil {
		types.HandleError(c, err, "media not found")
		return
	}
	if dto.IsActive != nil && !*dto.IsActive {
		if err := h.store.UpdateBanner(c.Request.Context(), b.ID, map[string]any{"is_active": false}); err != nil {
			types.HandleError(c, err, "banner not found")
			return
		}
	}
	created, err := h.store.FindBanner(c.Request.Context(), b.ID)
	if err != nil {
		types.HandleError(c, err, "banner not found")
		return
	}
	c.JSON(http.StatusCreated, created)
}

// UpdateBanner godoc
// @Summary Update banner
// @Tags CMS
// @Accept json
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param id path int true "Banner ID"
// @Param body body UpdateBannerDto true "Fields"
// @Success 200 {object} types.ApiResponse{data=Banner}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /cms/banners/{id} [patch]
func (h *Handler) UpdateBanner(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	var dto UpdateBannerDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	fields := map[string]any{}
	if dto.Title != nil {
		fields["title"] = *dto.Title
	}
	if dto.MediaID != nil {
		fields["media_id"] = *dto.MediaID
	}
	if dto.AltName != nil {
		fields["alt_name"] = *dto.AltName
	}
	if dto.LinkUrl != nil {
		fields["link_url"] = *dto.LinkUrl
	}
	if dto.DisplayOrder != nil {
		fields["display_order"] = *dto.DisplayOrder
	}
	if dto.IsActive != nil {
		fields["is_active"] = *dto.IsActive
	}
	if len(fields) > 0 {
		if err := h.store.UpdateBanner(c.Request.Context(), id, fields); err != nil {
			types.HandleError(c, err, "banner not found")
			return
		}
	}
	b, err := h.store.FindBanner(c.Request.Context(), id)
	if err != nil {
		types.HandleError(c, err, "banner not found")
		return
	}
	c.JSON(http.StatusOK, b)
}

// DeleteBanner godoc
// @Summary Delete banner
// @Tags CMS
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param id path int true "Banner ID"
// @Success 200 {object} types.ApiResponse{data=types.MessageResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /cms/banners/{id} [delete]
func (h *Handler) DeleteBanner(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	if err := h.store.DeleteBanner(c.Request.Context(), id); err != nil {
		types.HandleError(c, err, "banner not found")
		return
	}
	c.JSON(http.StatusOK, types.MessageResponse{Message: "deleted"})
}
