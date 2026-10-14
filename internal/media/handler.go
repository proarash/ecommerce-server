package media

import (
	"errors"
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/middleware"
	"github.com/proarash/ecommerce-server/internal/types"
)

const maxUploadBody = 10*MaxImageSize + 1<<20

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(public gin.IRouter, storekeeper gin.IRouter, uploader gin.IRouter) {
	public.GET("/media/:file", h.Serve)
	uploader.GET("/media", h.List)
	uploader.POST("/media/upload", h.Upload)
	storekeeper.PUT("/media/:id", h.Replace)
	storekeeper.DELETE("/media/:id", h.Delete)
}

func (h *Handler) fail(c *gin.Context, err error, notFound string) {
	if errors.Is(err, ErrFileTooLarge) || errors.Is(err, ErrUnsupportedImage) || errors.Is(err, ErrSingleImage) {
		types.BadRequest(c, err)
		return
	}
	types.HandleError(c, err, notFound)
}

func formFiles(c *gin.Context, field string) ([]*multipart.FileHeader, bool) {
	c.Request.Body = http.MaxBytesReader(c.Writer, c.Request.Body, maxUploadBody)
	form, err := c.MultipartForm()
	if err != nil {
		c.JSON(http.StatusRequestEntityTooLarge, types.ErrorResponse{Error: "invalid or too large upload"})
		return nil, false
	}
	files := form.File[field]
	if len(files) == 0 {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "at least one file is required in the " + field + " field"})
		return nil, false
	}
	return files, true
}

// List godoc
// @Summary List uploaded media
// @Description Newest first. Use the returned ids with POST /products/{id}/media/assign, banner media_id, and so on.
// @Tags Media
// @Produce json
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Param id query int false "Filter by media ID"
// @Param mime query string false "Filter by MIME type" Enums(image/webp, image/svg+xml)
// @Success 200 {object} types.ApiResponse{data=types.PaginatedResponse{items=[]Media}}
// @Security BearerAuth
// @Router /media [get]
func (h *Handler) List(c *gin.Context) {
	var q ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	items, total, err := h.service.List(c.Request.Context(), q)
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	c.JSON(http.StatusOK, types.PaginatedResponse{Items: items, Total: total, Page: q.Page, Limit: q.Limit})
}

// Upload godoc
// @Summary Upload images
// @Description Uploads images (webp, svg; max 2MB each). Without pid and type the images are only stored and returned, so their ids can be assigned later (e.g. POST /products/{id}/media/assign). With pid and type they are also assigned right away: 1 = product (one or more images, appended), 2 = category (exactly one image, replaces the current one), 3 = banner (exactly one image, replaces the current one). Assigning to products and categories requires the storekeeper role, banners the marketer role. Files are stored as <uuid>.<extension> and served from /media/{file}.
// @Tags Media
// @Accept multipart/form-data
// @Produce json
// @Param pid query int false "Product, category or banner ID; required together with type"
// @Param type query int false "Target type: 1 = product, 2 = category, 3 = banner; omit both to upload without assigning" Enums(1, 2, 3)
// @Param files formData file true "Image files; repeat the field for multiple product images, exactly one for a category or banner"
// @Success 201 {object} types.ApiResponse{data=[]Media}
// @Security BearerAuth
// @Router /media/upload [post]
func (h *Handler) Upload(c *gin.Context) {
	var q UploadQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	if q.Type != 0 {
		requiredRole := "storekeeper"
		if q.Type == TargetBanner {
			requiredRole = "marketer"
		}
		if p := middleware.GetAuth(c); p == nil || !(p.HasRole(middleware.RoleAdmin) || p.HasRole(requiredRole)) {
			c.AbortWithStatusJSON(http.StatusForbidden, types.ErrorResponse{Error: "forbidden"})
			return
		}
	}
	files, ok := formFiles(c, "files")
	if !ok {
		return
	}
	notFound := "product not found"
	switch q.Type {
	case TargetCategory:
		notFound = "category not found"
	case TargetBanner:
		notFound = "banner not found"
	}
	items, err := h.service.Upload(c.Request.Context(), q.Type, q.PID, files)
	if err != nil {
		h.fail(c, err, notFound)
		return
	}
	c.JSON(http.StatusCreated, items)
}

// Replace godoc
// @Summary Replace media image
// @Description Replaces the image file of an existing media record (webp, svg; max 2MB). The record keeps its ID and assignments, so every product or category using it shows the new image. The file gets a new <uuid>.<extension> name and the old file is deleted from disk.
// @Tags Media
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "Media ID"
// @Param file formData file true "Image file"
// @Success 200 {object} types.ApiResponse{data=Media}
// @Security BearerAuth
// @Router /media/{id} [put]
func (h *Handler) Replace(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	files, ok := formFiles(c, "file")
	if !ok {
		return
	}
	if len(files) != 1 {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "exactly one file is required in the file field"})
		return
	}
	m, err := h.service.Replace(c.Request.Context(), id, files[0])
	if err != nil {
		h.fail(c, err, "media not found")
		return
	}
	c.JSON(http.StatusOK, m)
}

// Delete godoc
// @Summary Delete media
// @Description Permanently deletes the media record and its file from disk, unassigning it from every product and category that uses it.
// @Tags Media
// @Produce json
// @Param id path int true "Media ID"
// @Success 200 {object} types.ApiResponse{data=types.MessageResponse}
// @Security BearerAuth
// @Router /media/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	if err := h.service.Delete(c.Request.Context(), id); err != nil {
		types.HandleError(c, err, "media not found")
		return
	}
	c.JSON(http.StatusOK, types.MessageResponse{Message: "deleted"})
}

// Serve godoc
// @Summary Serve uploaded image
// @Description Serves an uploaded image by its exact stored file name (<uuid>.<extension>). Any other name returns 404; directory listing is not available.
// @Tags Media
// @Produce image/webp,image/svg+xml
// @Param file path string true "Stored file name, e.g. 3f2b8c1e-9a4d-4e2f-8b1a-2c3d4e5f6a7b.webp"
// @Success 200 {file} binary
// @Router /media/{file} [get]
func (h *Handler) Serve(c *gin.Context) {
	path, err := h.service.Resolve(c.Param("file"))
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse{Error: "media not found"})
		return
	}
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Content-Security-Policy", "default-src 'none'; style-src 'unsafe-inline'; sandbox")
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.File(path)
}
