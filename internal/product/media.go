package product

import (
	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/types"
)

// AssignProductMedia godoc
// @Summary Assign existing media to product
// @Description Assigns already uploaded media to the product so multiple products can share the same image. Already assigned media are kept.
// @Tags Products
// @Accept json
// @Produce json
// @Param id path int true "Product ID"
// @Param body body AssignMediaDto true "Media IDs"
// @Success 200 {object} types.ApiResponse{data=Product}
// @Router /products/{id}/media/assign [post]
func (h *Handler) AssignProductMedia(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	var dto AssignMediaDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	if err := h.store.AttachMediaIDs(c.Request.Context(), id, dto.MediaIDs); err != nil {
		h.fail(c, err, "product not found")
		return
	}
	h.GetProduct(c)
}

// DetachProductMedia godoc
// @Summary Unassign media from product
// @Description Removes the media from this product only; the file stays available to other products sharing it.
// @Tags Products
// @Produce json
// @Param id path int true "Product ID"
// @Param mediaId path int true "Media ID"
// @Success 200 {object} types.ApiResponse{data=Product}
// @Router /products/{id}/media/{mediaId} [delete]
func (h *Handler) DetachProductMedia(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	mediaID, ok := types.ParamID(c, "mediaId")
	if !ok {
		return
	}
	if err := h.store.DetachMedia(c.Request.Context(), id, mediaID); err != nil {
		types.HandleError(c, err, "product or media not found")
		return
	}
	h.GetProduct(c)
}

// RemoveCategoryMedia godoc
// @Summary Remove category image
// @Tags Categories
// @Produce json
// @Param id path int true "Category ID"
// @Success 200 {object} types.ApiResponse{data=Category}
// @Router /categories/{id}/media [delete]
func (h *Handler) RemoveCategoryMedia(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	if err := h.store.SetCategoryMedia(c.Request.Context(), id, nil); err != nil {
		types.HandleError(c, err, "category not found")
		return
	}
	h.GetCategory(c)
}
