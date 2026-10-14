package media

import (
	"errors"
	"mime/multipart"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/types"
)

const maxUploadBody = 10*MaxImageSize + 1<<20

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(public gin.IRouter, storekeeper gin.IRouter) {
	public.GET("/media/:file", h.Serve)
	storekeeper.POST("/media/upload", h.Upload)
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

// Upload godoc
// @Summary Upload images for a product or category
// @Description Uploads images (jpeg, png, gif, webp; max 2MB each) and assigns them to the target selected by type: 1 = product (one or more images, appended), 2 = category (exactly one image, replaces the current one). Files are stored as <uuid>.<extension> and served from /media/{file}.
// @Tags Media
// @Accept multipart/form-data
// @Produce json
// @Param pid query int true "Product ID or Category ID, depending on type"
// @Param type query int true "Target type: 1 = product, 2 = category" Enums(1, 2)
// @Param files formData file true "Image files; repeat the field for multiple product images, exactly one for a category"
// @Success 201 {object} types.ApiResponse{data=[]Media}
// @Router /media/upload [post]
func (h *Handler) Upload(c *gin.Context) {
	var q UploadQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	files, ok := formFiles(c, "files")
	if !ok {
		return
	}
	notFound := "product not found"
	if q.Type == TargetCategory {
		notFound = "category not found"
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
// @Description Replaces the image file of an existing media record (jpeg, png, gif, webp; max 2MB). The record keeps its ID and assignments, so every product or category using it shows the new image. The file gets a new <uuid>.<extension> name and the old file is deleted from disk.
// @Tags Media
// @Accept multipart/form-data
// @Produce json
// @Param id path int true "Media ID"
// @Param file formData file true "Image file"
// @Success 200 {object} types.ApiResponse{data=Media}
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
// @Produce image/jpeg,image/png,image/gif,image/webp
// @Param file path string true "Stored file name, e.g. 3f2b8c1e-9a4d-4e2f-8b1a-2c3d4e5f6a7b.jpg"
// @Success 200 {file} binary
// @Router /media/{file} [get]
func (h *Handler) Serve(c *gin.Context) {
	path, err := h.service.Resolve(c.Param("file"))
	if err != nil {
		c.JSON(http.StatusNotFound, types.ErrorResponse{Error: "media not found"})
		return
	}
	c.Header("X-Content-Type-Options", "nosniff")
	c.Header("Cache-Control", "public, max-age=31536000, immutable")
	c.File(path)
}
