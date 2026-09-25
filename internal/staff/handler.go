package staff

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

func (h *Handler) RegisterRoutes(protected gin.IRouter) {
	g := protected.Group("/staff", middleware.RequireRoles(RoleStorekeeper, RoleAccountant, RoleMarketer, RoleSupport))
	g.GET("/me", h.Me)
	g.PATCH("/me", h.UpdateMe)
}

// Me godoc
// @Summary Get current staff profile
// @Tags Staff
// @Produce json
// @Security BearerAuth
// @Success 200 {object} types.ApiResponse{data=StaffUser}
// @Failure 401 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/staff/me [get]
func (h *Handler) Me(c *gin.Context) {
	u, err := h.store.FindByID(c.Request.Context(), middleware.GetAuth(c).UserID)
	if err != nil {
		types.HandleError(c, err, "staff not found")
		return
	}
	c.JSON(http.StatusOK, u)
}

// UpdateMe godoc
// @Summary Update current staff profile
// @Description default_avatar_id is a preset frontend avatar index (1..5); avatar_media_id references an uploaded MinIO media asset
// @Tags Staff
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body UpdateProfileDto true "Profile fields"
// @Success 200 {object} types.ApiResponse{data=StaffUser}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 401 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/staff/me [patch]
func (h *Handler) UpdateMe(c *gin.Context) {
	var dto UpdateProfileDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	fields := map[string]any{}
	if dto.Name != nil {
		fields["name"] = *dto.Name
	}
	if dto.AvatarMediaID != nil {
		fields["avatar_media_id"] = *dto.AvatarMediaID
	}
	if dto.DefaultAvatarID != nil {
		fields["default_avatar_id"] = *dto.DefaultAvatarID
	}
	id := middleware.GetAuth(c).UserID
	if len(fields) > 0 {
		if err := h.store.Update(c.Request.Context(), id, fields); err != nil {
			types.HandleError(c, err, "staff not found")
			return
		}
	}
	h.Me(c)
}
