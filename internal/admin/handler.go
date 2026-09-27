package admin

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/middleware"
	"github.com/proarash/ecommerce-server/internal/staff"
	"github.com/proarash/ecommerce-server/internal/types"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type Handler struct {
	db    *gorm.DB
	staff staff.Store
}

func NewHandler(db *gorm.DB, staffStore staff.Store) *Handler {
	return &Handler{db: db, staff: staffStore}
}

func (h *Handler) RegisterRoutes(admin gin.IRouter) {
	admin.POST("/staff", h.CreateStaff)
	admin.GET("/staff", h.ListStaff)
	admin.PATCH("/staff/:id/status", h.UpdateStaffStatus)
	admin.GET("/stats", h.Stats)
}

// CreateStaff godoc
// @Summary Create staff user
// @Description Creates storekeeper, accountant, marketer or support users. Creating another admin is not allowed. default_avatar_id is a preset frontend avatar index (1..5); avatar_media_id references an uploaded MinIO media asset.
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param body body CreateStaffDto true "Staff user"
// @Success 201 {object} types.ApiResponse{data=staff.StaffUser}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 409 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/admin/staff [post]
func (h *Handler) CreateStaff(c *gin.Context) {
	var dto CreateStaffDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
	if err != nil {
		types.HandleError(c, err, "")
		return
	}
	u := staff.StaffUser{Name: dto.Name, Mobile: dto.Mobile, Password: string(hash), Role: dto.Role, AvatarMediaID: dto.AvatarMediaID, DefaultAvatarID: dto.DefaultAvatarID, Status: true}
	if err := h.staff.Create(c.Request.Context(), &u); err != nil {
		types.HandleError(c, err, "media not found")
		return
	}
	c.JSON(http.StatusCreated, u)
}

// ListStaff godoc
// @Summary List staff users
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param role query string false "Role" Enums(admin, storekeeper, accountant, marketer, support)
// @Param page query int false "Page" default(1)
// @Param limit query int false "Limit" default(20)
// @Success 200 {object} types.ApiResponse{data=types.PaginatedResponse{items=[]staff.StaffUser}}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/admin/staff [get]
func (h *Handler) ListStaff(c *gin.Context) {
	var q StaffQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	items, total, err := h.staff.List(c.Request.Context(), q.Role, (q.Page-1)*q.Limit, q.Limit)
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	c.JSON(http.StatusOK, types.PaginatedResponse{Items: items, Total: total, Page: q.Page, Limit: q.Limit})
}

// UpdateStaffStatus godoc
// @Summary Enable or disable staff user
// @Tags Admin
// @Accept json
// @Produce json
// @Security BearerAuth
// @Param id path int true "Staff ID"
// @Param body body UpdateStaffStatusDto true "Status"
// @Success 200 {object} types.ApiResponse{data=staff.StaffUser}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/admin/staff/{id}/status [patch]
func (h *Handler) UpdateStaffStatus(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	var dto UpdateStaffStatusDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	if id == middleware.GetAuth(c).UserID {
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "cannot change your own status"})
		return
	}
	if err := h.staff.Update(c.Request.Context(), id, map[string]any{"status": *dto.Status}); err != nil {
		types.HandleError(c, err, "staff not found")
		return
	}
	u, err := h.staff.FindByID(c.Request.Context(), id)
	if err != nil {
		types.HandleError(c, err, "staff not found")
		return
	}
	c.JSON(http.StatusOK, u)
}

// Stats godoc
// @Summary System statistics
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Success 200 {object} types.ApiResponse{data=StatsResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 500 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/admin/stats [get]
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
	count("users", &s.Users)
	count("staff_users", &s.Staff)
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
