package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/middleware"
	"github.com/proarash/ecommerce-server/internal/types"
	"github.com/proarash/ecommerce-server/internal/user"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Handler struct {
	db    *gorm.DB
	users user.Store
}

func NewHandler(db *gorm.DB, userStore user.Store) *Handler {
	return &Handler{db: db, users: userStore}
}

func (h *Handler) RegisterRoutes(admin gin.IRouter) {
	admin.POST("/users", h.CreateUser)
	admin.GET("/users", h.ListUsers)
	admin.GET("/users/:id", h.GetUser)
	admin.PATCH("/users/:id/status", h.UpdateStatus)
	admin.PATCH("/users/:id/roles", h.UpdateRoles)
	admin.GET("/stats", h.Stats)
}

// CreateUser godoc
// @Summary Create user
// @Description Creates a user with the given roles (any of user, storekeeper, accountant, marketer, support; the user role is always included). Creating another admin is not allowed. default_avatar_id is a preset frontend avatar index (1..5); avatar_media_id references an uploaded media asset.
// @Tags Admin
// @Accept json
// @Produce json
// @Param body body CreateUserDto true "User"
// @Success 201 {object} types.ApiResponse{data=user.User}
// @Security BearerAuth
// @Router /admin/users [post]
func (h *Handler) CreateUser(c *gin.Context) {
	var dto CreateUserDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
	if err != nil {
		types.HandleError(c, err, "")
		return
	}
	u := user.User{Name: &dto.Name, Mobile: dto.Mobile, Password: string(hash), Roles: user.NewRoles(dto.Roles...), AvatarMediaID: dto.AvatarMediaID, DefaultAvatarID: dto.DefaultAvatarID, Status: true}
	if err := h.users.Create(c.Request.Context(), &u); err != nil {
		types.HandleError(c, err, "media not found")
		return
	}
	c.JSON(http.StatusCreated, u)
}

// ListUsers godoc
// @Summary List users
// @Description Lists all users. Filter by a role the user has, or set staff=true to list every user with at least one role besides "user".
// @Tags Admin
// @Produce json
// @Param role query string false "Role" Enums(user, admin, storekeeper, accountant, marketer, support)
// @Param staff query bool false "Only users with at least one staff role"
// @Param search query string false "Search in mobile or name"
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} types.ApiResponse{data=types.PaginatedResponse{items=[]user.User}}
// @Security BearerAuth
// @Router /admin/users [get]
func (h *Handler) ListUsers(c *gin.Context) {
	var q user.ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	items, total, err := h.users.List(c.Request.Context(), q)
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	c.JSON(http.StatusOK, types.PaginatedResponse{Items: items, Total: total, Page: q.Page, Limit: q.Limit})
}

// GetUser godoc
// @Summary Get user
// @Tags Admin
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} types.ApiResponse{data=user.User}
// @Security BearerAuth
// @Router /admin/users/{id} [get]
func (h *Handler) GetUser(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	u, err := h.users.FindByID(c.Request.Context(), id)
	if err != nil {
		types.HandleError(c, err, "user not found")
		return
	}
	c.JSON(http.StatusOK, u)
}

func (h *Handler) update(c *gin.Context, id uint, fields map[string]any) {
	if id == middleware.GetAuth(c).UserID {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "cannot change your own account here"})
		return
	}
	target, err := h.users.FindByID(c.Request.Context(), id)
	if err != nil {
		types.HandleError(c, err, "user not found")
		return
	}
	if target.Roles.Has(user.RoleAdmin) {
		c.JSON(http.StatusForbidden, types.ErrorResponse{Error: "admin accounts cannot be changed"})
		return
	}
	if err := h.users.Update(c.Request.Context(), id, fields); err != nil {
		types.HandleError(c, err, "user not found")
		return
	}
	h.GetUser(c)
}

// UpdateStatus godoc
// @Summary Enable or disable user
// @Description Disabled users cannot log in or refresh their token. Admin accounts and your own account cannot be changed.
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param body body UpdateStatusDto true "Status"
// @Success 200 {object} types.ApiResponse{data=user.User}
// @Security BearerAuth
// @Router /admin/users/{id}/status [patch]
func (h *Handler) UpdateStatus(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	var dto UpdateStatusDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	h.update(c, id, map[string]any{"status": *dto.Status})
}

// UpdateRoles godoc
// @Summary Set user roles
// @Description Replaces the roles of a user with the given list (any of user, storekeeper, accountant, marketer, support). The user role is always kept. The admin role cannot be granted or removed. The user's current token keeps the old roles until it is renewed.
// @Tags Admin
// @Accept json
// @Produce json
// @Param id path int true "User ID"
// @Param body body UpdateRolesDto true "Roles"
// @Success 200 {object} types.ApiResponse{data=user.User}
// @Security BearerAuth
// @Router /admin/users/{id}/roles [patch]
func (h *Handler) UpdateRoles(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	var dto UpdateRolesDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	h.update(c, id, map[string]any{"roles": user.NewRoles(dto.Roles...)})
}

// Stats godoc
// @Summary System statistics
// @Description users counts accounts that only have the user role; staff counts accounts with any other role.
// @Tags Admin
// @Produce json
// @Success 200 {object} types.ApiResponse{data=StatsResponse}
// @Security BearerAuth
// @Router /admin/stats [get]
func (h *Handler) Stats(c *gin.Context) {
	db := h.db.WithContext(c.Request.Context())
	var s StatsResponse
	var err error
	count := func(table string, dst *int64, where ...any) {
		if err != nil {
			return
		}
		q := db.Table(table).Where("deleted_at IS NULL")
		if len(where) > 0 {
			q = q.Where(where[0], where[1:]...)
		}
		err = q.Count(dst).Error
	}
	paid := []string{"paid", "processing", "delivered"}
	count("users", &s.Users, "jsonb_array_length(roles) = 1")
	count("users", &s.Staff, "jsonb_array_length(roles) > 1")
	count("products", &s.Products)
	count("orders", &s.Orders)
	count("orders", &s.PaidOrders, "status IN ?", paid)
	count("chat_rooms", &s.OpenChats, "status <> ?", "closed")
	count("blog_posts", &s.BlogPosts)
	if err == nil {
		err = db.Table("orders").Where("deleted_at IS NULL AND status IN ?", paid).Select("COALESCE(SUM(total_amount), 0)").Scan(&s.Revenue).Error
	}
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	c.JSON(http.StatusOK, s)
}
