package product

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/types"
)

type Handler struct {
	store Store
}

func NewHandler(store Store) *Handler {
	return &Handler{store: store}
}

func (h *Handler) RegisterRoutes(public gin.IRouter, storekeeper gin.IRouter) {
	public.GET("/products", h.ListProducts)
	public.GET("/products/:id", h.GetProduct)
	public.GET("/categories", h.ListCategories)
	public.GET("/categories/:id", h.GetCategory)

	storekeeper.POST("/products", h.CreateProduct)
	storekeeper.PATCH("/products/:id", h.UpdateProduct)
	storekeeper.DELETE("/products/:id", h.DeleteProduct)
	storekeeper.POST("/categories", h.CreateCategory)
	storekeeper.PATCH("/categories/:id", h.UpdateCategory)
	storekeeper.DELETE("/categories/:id", h.DeleteCategory)
}

func buildTree(all []Category) []CategoryTree {
	byParent := map[uint][]Category{}
	for _, c := range all {
		var pid uint
		if c.ParentID != nil {
			pid = *c.ParentID
		}
		byParent[pid] = append(byParent[pid], c)
	}
	var walk func(pid uint) []CategoryTree
	walk = func(pid uint) []CategoryTree {
		nodes := []CategoryTree{}
		for _, c := range byParent[pid] {
			n := CategoryTree{ID: c.ID, Name: c.Name, Slug: c.Slug, ParentID: c.ParentID, MediaID: c.MediaID, Children: walk(c.ID)}
			if c.Media != nil {
				n.MediaURL = &c.Media.URL
			}
			nodes = append(nodes, n)
		}
		return nodes
	}
	return walk(0)
}

// ListCategories godoc
// @Summary List categories as a tree
// @Tags Categories
// @Produce json
// @Success 200 {object} types.ApiResponse{data=[]CategoryTree}
// @Router /api/categories [get]
func (h *Handler) ListCategories(c *gin.Context) {
	all, err := h.store.ListCategories(c.Request.Context())
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	c.JSON(http.StatusOK, buildTree(all))
}

// GetCategory godoc
// @Summary Get category with direct children
// @Tags Categories
// @Produce json
// @Param id path int true "Category ID"
// @Success 200 {object} types.ApiResponse{data=Category}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/categories/{id} [get]
func (h *Handler) GetCategory(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	cat, err := h.store.FindCategory(c.Request.Context(), id)
	if err != nil {
		types.HandleError(c, err, "category not found")
		return
	}
	c.JSON(http.StatusOK, cat)
}

// CreateCategory godoc
// @Summary Create category
// @Tags Categories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateCategoryDto true "Category"
// @Success 201 {object} types.ApiResponse{data=Category}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 409 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/categories [post]
func (h *Handler) CreateCategory(c *gin.Context) {
	var dto CreateCategoryDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	cat := Category{Name: dto.Name, Slug: dto.Slug, ParentID: dto.ParentID, MediaID: dto.MediaID}
	if err := h.store.CreateCategory(c.Request.Context(), &cat); err != nil {
		types.HandleError(c, err, "parent or media not found")
		return
	}
	c.JSON(http.StatusCreated, cat)
}

// UpdateCategory godoc
// @Summary Update category
// @Tags Categories
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Category ID"
// @Param body body UpdateCategoryDto true "Fields"
// @Success 200 {object} types.ApiResponse{data=Category}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 409 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/categories/{id} [patch]
func (h *Handler) UpdateCategory(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	var dto UpdateCategoryDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	fields := map[string]any{}
	if dto.Name != nil {
		fields["name"] = *dto.Name
	}
	if dto.Slug != nil {
		fields["slug"] = *dto.Slug
	}
	if dto.ParentID != nil {
		fields["parent_id"] = *dto.ParentID
	}
	if dto.MediaID != nil {
		fields["media_id"] = *dto.MediaID
	}
	if len(fields) > 0 {
		if err := h.store.UpdateCategory(c.Request.Context(), id, fields); err != nil {
			if errors.Is(err, ErrCategoryCycle) {
				types.BadRequest(c, err)
				return
			}
			types.HandleError(c, err, "category not found")
			return
		}
	}
	h.GetCategory(c)
}

// DeleteCategory godoc
// @Summary Delete category
// @Description Soft-deletes the category; direct children become root categories
// @Tags Categories
// @Produce json
// @Security BearerAuth
// @Param id path int true "Category ID"
// @Success 200 {object} types.ApiResponse{data=types.MessageResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/categories/{id} [delete]
func (h *Handler) DeleteCategory(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	if err := h.store.DeleteCategory(c.Request.Context(), id); err != nil {
		types.HandleError(c, err, "category not found")
		return
	}
	c.JSON(http.StatusOK, types.MessageResponse{Message: "deleted"})
}

// ListProducts godoc
// @Summary List active products
// @Tags Products
// @Produce json
// @Param category_id query int false "Category ID"
// @Param search query string false "Title search"
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} types.ApiResponse{data=types.PaginatedResponse{items=[]Product}}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/products [get]
func (h *Handler) ListProducts(c *gin.Context) {
	var q ProductQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	items, total, err := h.store.ListProducts(c.Request.Context(), q, true)
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	c.JSON(http.StatusOK, types.PaginatedResponse{Items: items, Total: total, Page: q.Page, Limit: q.Limit})
}

// GetProduct godoc
// @Summary Get product
// @Tags Products
// @Produce json
// @Param id path int true "Product ID"
// @Success 200 {object} types.ApiResponse{data=Product}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/products/{id} [get]
func (h *Handler) GetProduct(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	p, err := h.store.FindProduct(c.Request.Context(), id)
	if err != nil {
		types.HandleError(c, err, "product not found")
		return
	}
	c.JSON(http.StatusOK, p)
}

// CreateProduct godoc
// @Summary Create product
// @Description Product title and SKU must be unique. media_ids attaches uploaded images/videos.
// @Tags Products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateProductDto true "Product"
// @Success 201 {object} types.ApiResponse{data=Product}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 409 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/products [post]
func (h *Handler) CreateProduct(c *gin.Context) {
	var dto CreateProductDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	p := Product{Title: dto.Title, Description: dto.Description, Price: dto.Price, SKU: dto.SKU, IsActive: true, CategoryID: dto.CategoryID}
	if dto.IsActive != nil {
		p.IsActive = *dto.IsActive
	}
	if err := h.store.CreateProduct(c.Request.Context(), &p, dto.MediaIDs); err != nil {
		types.HandleError(c, err, "category not found")
		return
	}
	if !p.IsActive {
		h.store.UpdateProduct(c.Request.Context(), p.ID, map[string]any{"is_active": false}, nil)
	}
	created, err := h.store.FindProduct(c.Request.Context(), p.ID)
	if err != nil {
		types.HandleError(c, err, "product not found")
		return
	}
	c.JSON(http.StatusCreated, created)
}

// UpdateProduct godoc
// @Summary Update product
// @Description media_ids, when provided, replaces the product media set
// @Tags Products
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Product ID"
// @Param body body UpdateProductDto true "Fields"
// @Success 200 {object} types.ApiResponse{data=Product}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 409 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/products/{id} [patch]
func (h *Handler) UpdateProduct(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	var dto UpdateProductDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	fields := map[string]any{}
	if dto.Title != nil {
		fields["title"] = *dto.Title
	}
	if dto.Description != nil {
		fields["description"] = *dto.Description
	}
	if dto.Price != nil {
		fields["price"] = *dto.Price
	}
	if dto.SKU != nil {
		fields["sku"] = *dto.SKU
	}
	if dto.IsActive != nil {
		fields["is_active"] = *dto.IsActive
	}
	if dto.CategoryID != nil {
		fields["category_id"] = *dto.CategoryID
	}
	if err := h.store.UpdateProduct(c.Request.Context(), id, fields, dto.MediaIDs); err != nil {
		types.HandleError(c, err, "product not found")
		return
	}
	h.GetProduct(c)
}

// DeleteProduct godoc
// @Summary Delete product
// @Tags Products
// @Produce json
// @Security BearerAuth
// @Param id path int true "Product ID"
// @Success 200 {object} types.ApiResponse{data=types.MessageResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/products/{id} [delete]
func (h *Handler) DeleteProduct(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	if err := h.store.DeleteProduct(c.Request.Context(), id); err != nil {
		types.HandleError(c, err, "product not found")
		return
	}
	c.JSON(http.StatusOK, types.MessageResponse{Message: "deleted"})
}
