package discount

import (
	"context"
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/middleware"
	"github.com/proarash/ecommerce-server/internal/types"
	"github.com/proarash/ecommerce-server/internal/user"
	"gorm.io/gorm"
)

var ErrOwnerNotFound = errors.New("no user registered with owner_mobile")

type Handler struct {
	store Store
	users user.Store
}

func NewHandler(store Store, users user.Store) *Handler {
	return &Handler{store: store, users: users}
}

func (h *Handler) RegisterRoutes(reader gin.IRouter, manager gin.IRouter, customer gin.IRouter) {
	reader.GET("/discounts", h.List)
	reader.GET("/discounts/:id", h.Get)

	manager.POST("/discounts", h.Create)
	manager.PATCH("/discounts/:id", h.Update)
	manager.DELETE("/discounts/:id", h.Delete)

	customer.POST("/user/discounts/check", h.Check)
}

func Fail(c *gin.Context, err error, notFound string) {
	switch {
	case errors.Is(err, ErrInvalidCode), errors.Is(err, ErrExhausted), errors.Is(err, ErrPercentRange),
		errors.Is(err, ErrMaxPriceValue), errors.Is(err, ErrOwnerNotFound):
		types.BadRequest(c, err)
	case errors.Is(err, ErrNotOwner):
		c.JSON(http.StatusForbidden, types.ErrorResponse{Error: err.Error()})
	default:
		types.HandleError(c, err, notFound)
	}
}

func (h *Handler) resolveOwner(ctx context.Context, mobile string) (*uint, error) {
	u, err := h.users.FindByMobile(ctx, mobile)
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, ErrOwnerNotFound
	}
	if err != nil {
		return nil, err
	}
	return &u.ID, nil
}

// Create godoc
// @Summary Generate discount code
// @Description Creates a discount code (admin, accountant, marketer). Omit code to auto-generate an 8 character uppercase code. max_price caps the discount and is only allowed for percent type (e.g. 30% up to 50000). When owner_mobile is set only the customer with that mobile can use the code.
// @Tags Discounts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateDiscountDto true "Discount"
// @Success 201 {object} types.ApiResponse{data=Discount}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 409 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/discounts [post]
func (h *Handler) Create(c *gin.Context) {
	var dto CreateDiscountDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	ctx := c.Request.Context()
	d := Discount{Code: NormalizeCode(dto.Code), Type: dto.Type, Value: dto.Value, MaxPrice: dto.MaxPrice, UseCount: dto.UseCount, Status: true, CreatedByID: middleware.GetAuth(c).UserID}
	if dto.Status != nil {
		d.Status = *dto.Status
	}
	if err := Validate(d); err != nil {
		Fail(c, err, "")
		return
	}
	if dto.OwnerMobile != nil {
		ownerID, err := h.resolveOwner(ctx, *dto.OwnerMobile)
		if err != nil {
			Fail(c, err, "")
			return
		}
		d.OwnerID = ownerID
	}
	if err := h.store.Create(ctx, &d); err != nil {
		Fail(c, err, "")
		return
	}
	res, err := h.store.FindByID(ctx, d.ID)
	if err != nil {
		Fail(c, err, "discount not found")
		return
	}
	c.JSON(http.StatusCreated, res)
}

// List godoc
// @Summary List discount codes
// @Description Available to admin, accountant, marketer and support (read-only)
// @Tags Discounts
// @Produce json
// @Security BearerAuth
// @Param code query string false "Code contains"
// @Param type query string false "Type" Enums(percent, value)
// @Param status query bool false "Status"
// @Param owner_id query int false "Owner user ID"
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} types.ApiResponse{data=types.PaginatedResponse{items=[]Discount}}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/discounts [get]
func (h *Handler) List(c *gin.Context) {
	var q ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	items, total, err := h.store.List(c.Request.Context(), q)
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	c.JSON(http.StatusOK, types.PaginatedResponse{Items: items, Total: total, Page: q.Page, Limit: q.Limit})
}

// Get godoc
// @Summary Get discount code
// @Description Available to admin, accountant, marketer and support (read-only)
// @Tags Discounts
// @Produce json
// @Security BearerAuth
// @Param id path int true "Discount ID"
// @Success 200 {object} types.ApiResponse{data=Discount}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/discounts/{id} [get]
func (h *Handler) Get(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	d, err := h.store.FindByID(c.Request.Context(), id)
	if err != nil {
		types.HandleError(c, err, "discount not found")
		return
	}
	c.JSON(http.StatusOK, d)
}

// Update godoc
// @Summary Update discount code
// @Description Updates a discount (admin, accountant, marketer). Send max_price 0 to remove the cap and owner_mobile "" to make the code public.
// @Tags Discounts
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Discount ID"
// @Param body body UpdateDiscountDto true "Fields to update"
// @Success 200 {object} types.ApiResponse{data=Discount}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/discounts/{id} [patch]
func (h *Handler) Update(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	var dto UpdateDiscountDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	ctx := c.Request.Context()
	d, err := h.store.FindByID(ctx, id)
	if err != nil {
		types.HandleError(c, err, "discount not found")
		return
	}
	fields := map[string]any{}
	if dto.Type != nil {
		d.Type = *dto.Type
		fields["type"] = d.Type
	}
	if dto.Value != nil {
		d.Value = *dto.Value
		fields["value"] = d.Value
	}
	if dto.MaxPrice != nil {
		d.MaxPrice = dto.MaxPrice
		if *dto.MaxPrice == 0 {
			d.MaxPrice = nil
		}
		fields["max_price"] = d.MaxPrice
	}
	if dto.UseCount != nil {
		fields["use_count"] = *dto.UseCount
	}
	if dto.Status != nil {
		fields["status"] = *dto.Status
	}
	if dto.OwnerMobile != nil {
		var ownerID *uint
		if *dto.OwnerMobile != "" {
			if ownerID, err = h.resolveOwner(ctx, *dto.OwnerMobile); err != nil {
				Fail(c, err, "")
				return
			}
		}
		fields["owner_id"] = ownerID
	}
	if err := Validate(d); err != nil {
		Fail(c, err, "")
		return
	}
	if len(fields) > 0 {
		if err := h.store.Update(ctx, id, fields); err != nil {
			Fail(c, err, "discount not found")
			return
		}
	}
	res, err := h.store.FindByID(ctx, id)
	if err != nil {
		types.HandleError(c, err, "discount not found")
		return
	}
	c.JSON(http.StatusOK, res)
}

// Delete godoc
// @Summary Delete discount code
// @Tags Discounts
// @Produce json
// @Security BearerAuth
// @Param id path int true "Discount ID"
// @Success 200 {object} types.ApiResponse{data=types.MessageResponse}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/discounts/{id} [delete]
func (h *Handler) Delete(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	if err := h.store.Delete(c.Request.Context(), id); err != nil {
		types.HandleError(c, err, "discount not found")
		return
	}
	c.JSON(http.StatusOK, types.MessageResponse{Message: "discount deleted"})
}

// Check godoc
// @Summary Check discount code
// @Description Validates a discount code for the current customer against an amount and returns the discount and payable amount without redeeming it
// @Tags User
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CheckDto true "Code and amount"
// @Success 200 {object} types.ApiResponse{data=CheckResponse}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/user/discounts/check [post]
func (h *Handler) Check(c *gin.Context) {
	var dto CheckDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	d, amount, err := h.store.Check(c.Request.Context(), dto.Code, middleware.GetAuth(c).UserID, dto.Amount)
	if err != nil {
		Fail(c, err, "")
		return
	}
	c.JSON(http.StatusOK, CheckResponse{Code: d.Code, Type: d.Type, Amount: dto.Amount, DiscountAmount: amount, PayableAmount: dto.Amount - amount})
}
