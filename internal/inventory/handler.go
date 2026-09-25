package inventory

import (
	"errors"
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

func (h *Handler) RegisterRoutes(storekeeper gin.IRouter) {
	g := storekeeper.Group("/inventory")
	g.GET("", h.List)
	g.POST("/inbound", h.Inbound)
	g.POST("/outbound", h.Outbound)
	g.GET("/logs/:productId", h.Logs)
}

// List godoc
// @Summary Current stock levels
// @Tags Inventory
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} types.ApiResponse{data=types.PaginatedResponse{items=[]InventoryStock}}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/inventory [get]
func (h *Handler) List(c *gin.Context) {
	var q types.Pagination
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	items, total, err := h.store.List(c.Request.Context(), q.Offset(), q.Limit)
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	c.JSON(http.StatusOK, types.PaginatedResponse{Items: items, Total: total, Page: q.Page, Limit: q.Limit})
}

// Inbound godoc
// @Summary Stock intake
// @Tags Inventory
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body InboundDto true "Inbound movement"
// @Success 200 {object} types.ApiResponse{data=InventoryStock}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/inventory/inbound [post]
func (h *Handler) Inbound(c *gin.Context) {
	var dto InboundDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	stock, err := h.store.Inbound(c.Request.Context(), dto, middleware.GetAuth(c).UserID)
	if err != nil {
		types.HandleError(c, err, "product not found")
		return
	}
	c.JSON(http.StatusOK, stock)
}

// Outbound godoc
// @Summary Stock dispatch or reduction
// @Tags Inventory
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body OutboundDto true "Outbound movement"
// @Success 200 {object} types.ApiResponse{data=InventoryStock}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 409 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/inventory/outbound [post]
func (h *Handler) Outbound(c *gin.Context) {
	var dto OutboundDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	stock, err := h.store.Outbound(c.Request.Context(), dto, middleware.GetAuth(c).UserID)
	if err != nil {
		if errors.Is(err, ErrInsufficientStock) {
			c.JSON(http.StatusConflict, types.ErrorResponse{Error: err.Error()})
			return
		}
		types.HandleError(c, err, "product not found")
		return
	}
	c.JSON(http.StatusOK, stock)
}

// Logs godoc
// @Summary Stock movement audit logs
// @Tags Inventory
// @Produce json
// @Security BearerAuth
// @Param productId path int true "Product ID"
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} types.ApiResponse{data=types.PaginatedResponse{items=[]InventoryLog}}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/inventory/logs/{productId} [get]
func (h *Handler) Logs(c *gin.Context) {
	id, ok := types.ParamID(c, "productId")
	if !ok {
		return
	}
	var q types.Pagination
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	items, total, err := h.store.Logs(c.Request.Context(), id, q.Offset(), q.Limit)
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	c.JSON(http.StatusOK, types.PaginatedResponse{Items: items, Total: total, Page: q.Page, Limit: q.Limit})
}
