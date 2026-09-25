package media

import (
	"errors"
	"fmt"
	"net/http"
	"path/filepath"
	"strings"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/types"
)

const maxUploadSize = 100 << 20

type Handler struct {
	store  Store
	client *Client
}

func NewHandler(store Store, client *Client) *Handler {
	return &Handler{store: store, client: client}
}

func (h *Handler) RegisterRoutes(protected gin.IRouter) {
	g := protected.Group("/media")
	g.POST("/upload", h.Upload)
	g.GET("/:id", h.Get)
}

func detectType(mime string) string {
	switch {
	case strings.HasPrefix(mime, "image/"):
		return TypeImage
	case strings.HasPrefix(mime, "video/"):
		return TypeVideo
	default:
		return TypeDocument
	}
}

// Upload godoc
// @Summary Upload media
// @Description Uploads an image, video or document to MinIO and stores its metadata
// @Tags Media
// @Accept multipart/form-data
// @Produce json
// @Security BearerAuth
// @Param file formData file true "Media file"
// @Success 201 {object} types.ApiResponse{data=UploadResponse}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 401 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 503 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/media/upload [post]
func (h *Handler) Upload(c *gin.Context) {
	fh, err := c.FormFile("file")
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "file is required"})
		return
	}
	if fh.Size > maxUploadSize {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "file too large"})
		return
	}
	f, err := fh.Open()
	if err != nil {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: err.Error()})
		return
	}
	defer f.Close()

	mime := fh.Header.Get("Content-Type")
	if mime == "" {
		mime = "application/octet-stream"
	}
	mediaType := detectType(mime)
	object := fmt.Sprintf("%s/%d%s", mediaType, time.Now().UnixNano(), strings.ToLower(filepath.Ext(fh.Filename)))

	url, err := h.client.Upload(c.Request.Context(), object, f, fh.Size, mime)
	if err != nil {
		if errors.Is(err, ErrStorageUnavailable) {
			c.JSON(http.StatusServiceUnavailable, types.ErrorResponse{Error: err.Error()})
			return
		}
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: err.Error()})
		return
	}

	m := Media{URL: url, MediaType: mediaType, MimeType: mime, FileSize: fh.Size, FileName: fh.Filename}
	if err := h.store.Create(c.Request.Context(), &m); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusCreated, UploadResponse{ID: m.ID, URL: m.URL, MediaType: m.MediaType, MimeType: m.MimeType, FileSize: m.FileSize, FileName: m.FileName})
}

// Get godoc
// @Summary Get media metadata
// @Tags Media
// @Produce json
// @Security BearerAuth
// @Param id path int true "Media ID"
// @Success 200 {object} types.ApiResponse{data=Media}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/media/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	m, err := h.store.FindByID(c.Request.Context(), id)
	if err != nil {
		types.HandleError(c, err, "media not found")
		return
	}
	c.JSON(http.StatusOK, m)
}
