package finance

import (
	"fmt"
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

func (h *Handler) RegisterRoutes(accountant gin.IRouter, customer gin.IRouter, support gin.IRouter) {
	accountant.GET("/finance/reports", h.Reports)
	accountant.POST("/finance/preinvoices", h.CreatePreInvoice)
	accountant.GET("/finance/preinvoices", h.ListPreInvoices)

	customer.GET("/user/orders", h.UserOrders)
	customer.GET("/user/orders/:id", h.UserOrder)
	customer.GET("/user/preinvoices", h.UserPreInvoices)

	support.GET("/orders", h.SupportOrders)
	support.GET("/preinvoices", h.SupportPreInvoices)
}

func (h *Handler) listOrders(c *gin.Context, q ListQuery) {
	items, total, err := h.store.ListOrders(c.Request.Context(), q)
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	c.JSON(http.StatusOK, types.PaginatedResponse{Items: items, Total: total, Page: q.Page, Limit: q.Limit})
}

func (h *Handler) listPreInvoices(c *gin.Context, q ListQuery) {
	items, total, err := h.store.ListPreInvoices(c.Request.Context(), q)
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	c.JSON(http.StatusOK, types.PaginatedResponse{Items: items, Total: total, Page: q.Page, Limit: q.Limit})
}

// Reports godoc
// @Summary Sales report
// @Description Paid, processing and delivered orders aggregated per period bucket
// @Tags Finance
// @Produce json
// @Security BearerAuth
// @Param period query string false "Bucket size" Enums(daily, weekly, monthly, annually) default(daily)
// @Param from query string false "From date (YYYY-MM-DD)"
// @Param to query string false "To date inclusive (YYYY-MM-DD)"
// @Success 200 {object} types.ApiResponse{data=ReportResponse}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/finance/reports [get]
func (h *Handler) Reports(c *gin.Context) {
	var q ReportQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	res, err := h.store.Report(c.Request.Context(), q)
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	c.JSON(http.StatusOK, res)
}

// CreatePreInvoice godoc
// @Summary Create manual pre-invoice
// @Description Issues a pre-invoice for an order (amount defaults to the order total) and notifies the customer
// @Tags Finance
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreatePreInvoiceDto true "Pre-invoice"
// @Success 201 {object} types.ApiResponse{data=PreInvoice}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/finance/preinvoices [post]
func (h *Handler) CreatePreInvoice(c *gin.Context) {
	var dto CreatePreInvoiceDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	p := middleware.GetAuth(c)
	pre, err := h.store.CreatePreInvoice(c.Request.Context(), dto, fmt.Sprintf("staff:%d", p.UserID))
	if err != nil {
		types.HandleError(c, err, "order not found")
		return
	}
	c.JSON(http.StatusCreated, pre)
}

// ListPreInvoices godoc
// @Summary List pre-invoices
// @Tags Finance
// @Produce json
// @Security BearerAuth
// @Param status query string false "Status" Enums(issued, paid, cancelled)
// @Param user_id query int false "User ID"
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} types.ApiResponse{data=types.PaginatedResponse{items=[]PreInvoice}}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/finance/preinvoices [get]
func (h *Handler) ListPreInvoices(c *gin.Context) {
	var q ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	h.listPreInvoices(c, q)
}

// UserOrders godoc
// @Summary Customer orders
// @Tags User
// @Produce json
// @Security BearerAuth
// @Param status query string false "Status"
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} types.ApiResponse{data=types.PaginatedResponse{items=[]Order}}
// @Failure 401 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/user/orders [get]
func (h *Handler) UserOrders(c *gin.Context) {
	var q ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	q.UserID = middleware.GetAuth(c).UserID
	h.listOrders(c, q)
}

// UserOrder godoc
// @Summary Customer order details
// @Tags User
// @Produce json
// @Security BearerAuth
// @Param id path int true "Order ID"
// @Success 200 {object} types.ApiResponse{data=Order}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/user/orders/{id} [get]
func (h *Handler) UserOrder(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	o, err := h.store.FindUserOrder(c.Request.Context(), id, middleware.GetAuth(c).UserID)
	if err != nil {
		types.HandleError(c, err, "order not found")
		return
	}
	c.JSON(http.StatusOK, o)
}

// UserPreInvoices godoc
// @Summary Customer pre-invoices
// @Tags User
// @Produce json
// @Security BearerAuth
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} types.ApiResponse{data=types.PaginatedResponse{items=[]PreInvoice}}
// @Failure 401 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/user/preinvoices [get]
func (h *Handler) UserPreInvoices(c *gin.Context) {
	var q ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	q.UserID = middleware.GetAuth(c).UserID
	h.listPreInvoices(c, q)
}

// SupportOrders godoc
// @Summary Read-only customer orders
// @Tags Support
// @Produce json
// @Security BearerAuth
// @Param status query string false "Status"
// @Param user_id query int false "User ID"
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} types.ApiResponse{data=types.PaginatedResponse{items=[]Order}}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/support/orders [get]
func (h *Handler) SupportOrders(c *gin.Context) {
	var q ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	h.listOrders(c, q)
}

// SupportPreInvoices godoc
// @Summary Read-only customer pre-invoices
// @Tags Support
// @Produce json
// @Security BearerAuth
// @Param status query string false "Status"
// @Param user_id query int false "User ID"
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} types.ApiResponse{data=types.PaginatedResponse{items=[]PreInvoice}}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/support/preinvoices [get]
func (h *Handler) SupportPreInvoices(c *gin.Context) {
	var q ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	h.listPreInvoices(c, q)
}
