package cart

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/finance"
	"github.com/proarash/ecommerce-server/internal/inventory"
	"github.com/proarash/ecommerce-server/internal/middleware"
	"github.com/proarash/ecommerce-server/internal/types"
)

type Handler struct {
	store Store
}

func NewHandler(store Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) RegisterRoutes(customer gin.IRouter) {
	g := customer.Group("/cart")
	g.GET("", h.Get)
	g.POST("/items", h.AddItem)
	g.PATCH("/items/:id", h.UpdateItem)
	g.DELETE("/items/:id", h.RemoveItem)
	g.POST("/checkout", h.Checkout)
}

func (h *Handler) fail(c *gin.Context, err error, notFound string) {
	if errors.Is(err, ErrProductUnavailable) || errors.Is(err, ErrEmptyCart) {
		types.BadRequest(c, err)
		return
	}
	if errors.Is(err, inventory.ErrInsufficientStock) {
		c.JSON(http.StatusConflict, types.ErrorResponse{Error: err.Error()})
		return
	}
	types.HandleError(c, err, notFound)
}

// Get godoc
// @Summary Get current cart
// @Tags Cart
// @Produce json
// @Security BearerAuth
// @Success 200 {object} types.ApiResponse{data=CartResponse}
// @Failure 401 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/cart [get]
func (h *Handler) Get(c *gin.Context) {
	cart, err := h.store.Get(c.Request.Context(), middleware.GetAuth(c).UserID)
	if err != nil {
		h.fail(c, err, "cart not found")
		return
	}
	res := CartResponse{Cart: cart}
	for _, it := range cart.Items {
		if it.Product != nil {
			res.TotalAmount += it.Product.Price * float64(it.Quantity)
		}
	}
	c.JSON(http.StatusOK, res)
}

// AddItem godoc
// @Summary Add product to cart
// @Description Adds the product or increases its quantity if already present
// @Tags Cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body AddItemDto true "Item"
// @Success 200 {object} types.ApiResponse{data=CartResponse}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/cart/items [post]
func (h *Handler) AddItem(c *gin.Context) {
	var dto AddItemDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	if err := h.store.AddItem(c.Request.Context(), middleware.GetAuth(c).UserID, dto); err != nil {
		h.fail(c, err, "product not found")
		return
	}
	h.Get(c)
}

// UpdateItem godoc
// @Summary Update cart item quantity
// @Tags Cart
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Cart item ID"
// @Param body body UpdateItemDto true "Quantity"
// @Success 200 {object} types.ApiResponse{data=CartResponse}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/cart/items/{id} [patch]
func (h *Handler) UpdateItem(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	var dto UpdateItemDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	if err := h.store.UpdateItem(c.Request.Context(), middleware.GetAuth(c).UserID, id, dto.Quantity); err != nil {
		h.fail(c, err, "cart item not found")
		return
	}
	h.Get(c)
}

// RemoveItem godoc
// @Summary Remove item from cart
// @Tags Cart
// @Produce json
// @Security BearerAuth
// @Param id path int true "Cart item ID"
// @Success 200 {object} types.ApiResponse{data=CartResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/cart/items/{id} [delete]
func (h *Handler) RemoveItem(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	if err := h.store.RemoveItem(c.Request.Context(), middleware.GetAuth(c).UserID, id); err != nil {
		h.fail(c, err, "cart item not found")
		return
	}
	h.Get(c)
}

// Checkout godoc
// @Summary Checkout cart
// @Description Converts the cart into a pending order, reserves stock (outbound inventory movement), issues a system pre-invoice and sends invoice/pre-invoice bot messages
// @Tags Cart
// @Produce json
// @Security BearerAuth
// @Success 201 {object} types.ApiResponse{data=finance.Order}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 401 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 409 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/cart/checkout [post]
func (h *Handler) Checkout(c *gin.Context) {
	var order finance.Order
	order, err := h.store.Checkout(c.Request.Context(), middleware.GetAuth(c).UserID)
	if err != nil {
		h.fail(c, err, "cart not found")
		return
	}
	c.JSON(http.StatusCreated, order)
}
