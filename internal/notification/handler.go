package notification

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/middleware"
	"github.com/proarash/ecommerce-server/internal/types"
)

type Handler struct {
	store   Store
	service *Service
}

func NewHandler(store Store, service *Service) *Handler {
	return &Handler{store: store, service: service}
}

func (h *Handler) RegisterRoutes(admin gin.IRouter, customer gin.IRouter) {
	g := admin.Group("/notifications")
	g.POST("", h.Create)
	g.GET("", h.List)
	g.GET("/:id", h.Get)
	g.PATCH("/:id", h.Update)
	g.DELETE("/:id", h.Delete)
	customer.GET("/user/notifications", h.UserList)
	customer.PATCH("/user/notifications/:id/read", h.MarkRead)
}

// Create godoc
// @Summary Create and dispatch notification
// @Description Stores the notification and dispatches it via Telegram when channel is telegram or both. Omit user_id to broadcast.
// @Tags Admin Notifications
// @Accept json
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param body body CreateNotificationDto true "Notification"
// @Success 201 {object} types.ApiResponse{data=Notification}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /admin/notifications [post]
func (h *Handler) Create(c *gin.Context) {
	var dto CreateNotificationDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	n := Notification{UserID: dto.UserID, Title: dto.Title, Message: dto.Message, Type: dto.Type, Channel: dto.Channel}
	if err := h.service.Dispatch(c.Request.Context(), &n); err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	c.JSON(http.StatusCreated, n)
}

// List godoc
// @Summary List notifications
// @Tags Admin Notifications
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param user_id query int false "Filter by user ID"
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} types.ApiResponse{data=types.PaginatedResponse{items=[]Notification}}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /admin/notifications [get]
func (h *Handler) List(c *gin.Context) {
	var q ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	items, total, err := h.store.List(c.Request.Context(), q.UserID, (q.Page-1)*q.Limit, q.Limit)
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	c.JSON(http.StatusOK, types.PaginatedResponse{Items: items, Total: total, Page: q.Page, Limit: q.Limit})
}

// Get godoc
// @Summary Get notification
// @Tags Admin Notifications
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param id path int true "Notification ID"
// @Success 200 {object} types.ApiResponse{data=Notification}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /admin/notifications/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	n, err := h.store.FindByID(c.Request.Context(), id)
	if err != nil {
		types.HandleError(c, err, "notification not found")
		return
	}
	c.JSON(http.StatusOK, n)
}

// Update godoc
// @Summary Update notification
// @Tags Admin Notifications
// @Accept json
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param id path int true "Notification ID"
// @Param body body UpdateNotificationDto true "Fields"
// @Success 200 {object} types.ApiResponse{data=Notification}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /admin/notifications/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	var dto UpdateNotificationDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	fields := map[string]any{}
	if dto.Title != nil {
		fields["title"] = *dto.Title
	}
	if dto.Message != nil {
		fields["message"] = *dto.Message
	}
	if dto.Type != nil {
		fields["type"] = *dto.Type
	}
	if dto.Channel != nil {
		fields["channel"] = *dto.Channel
	}
	if dto.IsRead != nil {
		fields["is_read"] = *dto.IsRead
	}
	if len(fields) > 0 {
		if err := h.store.Update(c.Request.Context(), id, fields); err != nil {
			types.HandleError(c, err, "notification not found")
			return
		}
	}
	h.Get(c)
}

// Delete godoc
// @Summary Delete notification
// @Tags Admin Notifications
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param id path int true "Notification ID"
// @Success 200 {object} types.ApiResponse{data=types.MessageResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /admin/notifications/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	if err := h.store.Delete(c.Request.Context(), id); err != nil {
		types.HandleError(c, err, "notification not found")
		return
	}
	c.JSON(http.StatusOK, types.MessageResponse{Message: "deleted"})
}

// UserList godoc
// @Summary Customer in-app notifications
// @Description Personal and broadcast in-app notifications
// @Tags User
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} types.ApiResponse{data=types.PaginatedResponse{items=[]Notification}}
// @Failure 401 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /user/notifications [get]
func (h *Handler) UserList(c *gin.Context) {
	var q types.Pagination
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	items, total, err := h.store.ListForUser(c.Request.Context(), middleware.GetAuth(c).UserID, q.Offset(), q.Limit)
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	c.JSON(http.StatusOK, types.PaginatedResponse{Items: items, Total: total, Page: q.Page, Limit: q.Limit})
}

// MarkRead godoc
// @Summary Mark notification as read
// @Tags User
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param id path int true "Notification ID"
// @Success 200 {object} types.ApiResponse{data=types.MessageResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /user/notifications/{id}/read [patch]
func (h *Handler) MarkRead(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	if err := h.store.MarkRead(c.Request.Context(), id, middleware.GetAuth(c).UserID); err != nil {
		types.HandleError(c, err, "notification not found")
		return
	}
	c.JSON(http.StatusOK, types.MessageResponse{Message: "updated"})
}
