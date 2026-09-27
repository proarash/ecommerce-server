package chat

import (
	"context"
	"log"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/gorilla/websocket"
	"github.com/proarash/ecommerce-server/internal/middleware"
	"github.com/proarash/ecommerce-server/internal/staff"
	"github.com/proarash/ecommerce-server/internal/types"
	"github.com/proarash/ecommerce-server/pkg/token"
)

var upgrader = websocket.Upgrader{
	ReadBufferSize:  1024,
	WriteBufferSize: 1024,
	CheckOrigin:     func(r *http.Request) bool { return true },
}

type RoomStatusDto struct {
	Status string `json:"status" binding:"required,oneof=open in_progress closed" enums:"open,in_progress,closed"`
}

type SendMessageDto struct {
	Message string `json:"message" binding:"required,min=1,max=4000"`
}

type RoomsQuery struct {
	types.Pagination
	Status string `form:"status" binding:"omitempty,oneof=open in_progress closed"`
}

type Handler struct {
	hub     *Hub
	service *Service
}

func NewHandler(hub *Hub, service *Service) *Handler {
	return &Handler{hub: hub, service: service}
}

func (h *Handler) RegisterRoutes(ws gin.IRouter, customer gin.IRouter, support gin.IRouter) {
	ws.GET("/chat", h.ServeWS)
	customer.GET("/user/chat/messages", h.UserMessages)
	customer.POST("/user/chat/messages", h.UserSend)
	support.GET("/chat/rooms", h.Rooms)
	support.GET("/chat/rooms/:id/messages", h.RoomMessages)
	support.POST("/chat/rooms/:id/messages", h.SupportSend)
	support.PATCH("/chat/rooms/:id/status", h.UpdateRoomStatus)
}

// ServeWS godoc
// @Summary Real-time support chat WebSocket
// @Description Upgrades to WebSocket. Authenticates via the staff_access_token or user_access_token httpOnly cookie (select with ?user_type=). Customers send {"message":"..."}; support agents send {"room_id":1,"message":"..."}. Server pushes ChatMessage JSON frames.
// @Tags Chat
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Success 101 {object} ChatMessage
// @Failure 401 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /ws/chat [get]
func (h *Handler) ServeWS(c *gin.Context) {
	p := middleware.GetAuth(c)
	client := &Client{hub: h.hub, service: h.service, send: make(chan []byte, 256), senderID: p.UserID}
	switch {
	case p.UserType == token.UserTypeCustomer:
		room, created, err := h.service.EnsureRoom(c.Request.Context(), p.UserID)
		if err != nil {
			c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: err.Error()})
			return
		}
		if created && h.service.OnRoomCreated != nil {
			go h.service.OnRoomCreated(context.Background(), room)
		}
		client.roomID = room.ID
	case p.UserType == token.UserTypeStaff && (p.Role == staff.RoleSupport || p.Role == staff.RoleAdmin):
		client.isSupport = true
	default:
		c.JSON(http.StatusForbidden, types.ErrorResponse{Error: "forbidden"})
		return
	}
	conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
	if err != nil {
		log.Println("chat: upgrade:", err)
		return
	}
	client.conn = conn
	h.hub.register(client)
	go client.writePump()
	go client.readPump()
}

// UserMessages godoc
// @Summary Customer chat inbox
// @Description Chat history of the current customer including automated bot messages
// @Tags Chat
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} types.ApiResponse{data=types.PaginatedResponse{items=[]ChatMessage}}
// @Failure 401 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /user/chat/messages [get]
func (h *Handler) UserMessages(c *gin.Context) {
	var q types.Pagination
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	items, total, err := h.service.UserMessages(c.Request.Context(), middleware.GetAuth(c).UserID, q.Offset(), q.Limit)
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	c.JSON(http.StatusOK, types.PaginatedResponse{Items: items, Total: total, Page: q.Page, Limit: q.Limit})
}

// UserSend godoc
// @Summary Send chat message as customer
// @Description REST fallback for sending a message to support; opens a room if none is active
// @Tags Chat
// @Accept json
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param body body SendMessageDto true "Message"
// @Success 201 {object} types.ApiResponse{data=ChatMessage}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 401 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /user/chat/messages [post]
func (h *Handler) UserSend(c *gin.Context) {
	var dto SendMessageDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	uid := middleware.GetAuth(c).UserID
	room, created, err := h.service.EnsureRoom(c.Request.Context(), uid)
	if err != nil {
		types.HandleError(c, err, "room not found")
		return
	}
	if created && h.service.OnRoomCreated != nil {
		go h.service.OnRoomCreated(context.Background(), room)
	}
	msg, err := h.service.UserMessage(c.Request.Context(), room.ID, uid, dto.Message)
	if err != nil {
		types.HandleError(c, err, "room not found")
		return
	}
	c.JSON(http.StatusCreated, msg)
}

// Rooms godoc
// @Summary List support chat rooms
// @Tags Support
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param status query string false "Room status" Enums(open, in_progress, closed)
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} types.ApiResponse{data=types.PaginatedResponse{items=[]ChatRoom}}
// @Failure 401 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /support/chat/rooms [get]
func (h *Handler) Rooms(c *gin.Context) {
	var q RoomsQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	items, total, err := h.service.Rooms(c.Request.Context(), q.Status, q.Offset(), q.Limit)
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	c.JSON(http.StatusOK, types.PaginatedResponse{Items: items, Total: total, Page: q.Page, Limit: q.Limit})
}

// RoomMessages godoc
// @Summary List messages of a chat room
// @Tags Support
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param id path int true "Room ID"
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} types.ApiResponse{data=types.PaginatedResponse{items=[]ChatMessage}}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /support/chat/rooms/{id}/messages [get]
func (h *Handler) RoomMessages(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	var q types.Pagination
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	items, total, err := h.service.RoomMessages(c.Request.Context(), id, q.Offset(), q.Limit)
	if err != nil {
		types.HandleError(c, err, "room not found")
		return
	}
	c.JSON(http.StatusOK, types.PaginatedResponse{Items: items, Total: total, Page: q.Page, Limit: q.Limit})
}

// SupportSend godoc
// @Summary Reply to a chat room as support
// @Tags Support
// @Accept json
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param id path int true "Room ID"
// @Param body body SendMessageDto true "Message"
// @Success 201 {object} types.ApiResponse{data=ChatMessage}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /support/chat/rooms/{id}/messages [post]
func (h *Handler) SupportSend(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	var dto SendMessageDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	msg, err := h.service.SupportReply(c.Request.Context(), id, middleware.GetAuth(c).UserID, dto.Message)
	if err != nil {
		types.HandleError(c, err, "room not found")
		return
	}
	c.JSON(http.StatusCreated, msg)
}

// UpdateRoomStatus godoc
// @Summary Update chat room status
// @Tags Support
// @Accept json
// @Produce json
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Security BearerAuth
// @Param id path int true "Room ID"
// @Param body body RoomStatusDto true "Status"
// @Success 200 {object} types.ApiResponse{data=types.MessageResponse}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /support/chat/rooms/{id}/status [patch]
func (h *Handler) UpdateRoomStatus(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	var dto RoomStatusDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	if err := h.service.UpdateRoomStatus(c.Request.Context(), id, dto.Status); err != nil {
		types.HandleError(c, err, "room not found")
		return
	}
	c.JSON(http.StatusOK, types.MessageResponse{Message: "updated"})
}
