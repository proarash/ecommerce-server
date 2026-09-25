package user

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/middleware"
	"github.com/proarash/ecommerce-server/internal/types"
)

type Handler struct {
	service *Service
}

func NewHandler(service *Service) *Handler {
	return &Handler{service: service}
}

func (h *Handler) RegisterRoutes(customer gin.IRouter) {
	g := customer.Group("/user")
	g.GET("/profile", h.GetProfile)
	g.PATCH("/profile", h.UpdateProfile)
}

// GetProfile godoc
// @Summary Get customer profile
// @Tags User
// @Produce json
// @Security BearerAuth
// @Success 200 {object} types.ApiResponse{data=User}
// @Failure 401 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/user/profile [get]
func (h *Handler) GetProfile(c *gin.Context) {
	u, err := h.service.Profile(c.Request.Context(), middleware.GetAuth(c).UserID)
	if err != nil {
		types.HandleError(c, err, "user not found")
		return
	}
	c.JSON(http.StatusOK, u)
}

// UpdateProfile godoc
// @Summary Update customer profile
// @Description Updates name, mobile, address, coordinates and telegram chat id (customers have no avatar)
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body UpdateProfileDto true "Profile fields"
// @Success 200 {object} types.ApiResponse{data=User}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 401 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 409 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/user/profile [patch]
func (h *Handler) UpdateProfile(c *gin.Context) {
	var dto UpdateProfileDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	u, err := h.service.UpdateProfile(c.Request.Context(), middleware.GetAuth(c).UserID, dto)
	if err != nil {
		types.HandleError(c, err, "user not found")
		return
	}
	c.JSON(http.StatusOK, u)
}
