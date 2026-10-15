package admin

import (
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/export"
	"github.com/proarash/ecommerce-server/internal/types"
	"github.com/proarash/ecommerce-server/internal/user"
)

var roleLabels = map[string]string{
	user.RoleAdmin:       "مدیر",
	user.RoleStorekeeper: "انباردار",
	user.RoleAccountant:  "حسابدار",
	user.RoleMarketer:    "بازاریاب",
	user.RoleSupport:     "پشتیبان",
	"user":               "کاربر",
}

// ExportUsers godoc
// @Summary Export users as excel
// @Description Same filters as GET /admin/users, without paging
// @Tags Admin
// @Produce application/vnd.openxmlformats-officedocument.spreadsheetml.sheet
// @Param role query string false "Role" Enums(user, admin, storekeeper, accountant, marketer, support)
// @Param staff query bool false "Only users with at least one staff role"
// @Param search query string false "Search in mobile or name"
// @Success 200 {file} file
// @Security BearerAuth
// @Router /admin/users/export [get]
func (h *Handler) ExportUsers(c *gin.Context) {
	var q user.ListQuery
	if err := c.ShouldBindQuery(&q); err != nil {
		types.BadRequest(c, err)
		return
	}
	q.Page, q.Limit = 1, export.MaxRows
	items, _, err := h.users.List(c.Request.Context(), q)
	if err != nil {
		types.HandleError(c, err, "not found")
		return
	}
	rows := make([][]any, 0, len(items))
	for _, u := range items {
		name := ""
		if u.Name != nil {
			name = *u.Name
		}
		roles := make([]string, 0, len(u.Roles))
		for _, r := range u.Roles {
			if label, ok := roleLabels[r]; ok {
				roles = append(roles, label)
			} else {
				roles = append(roles, r)
			}
		}
		state := "فعال"
		if !u.Status {
			state = "غیرفعال"
		}
		rows = append(rows, []any{u.ID, name, u.Mobile, strings.Join(roles, "، "), state, export.FormatTime(u.CreatedAt)})
	}
	export.Send(c, export.Filename("users"), export.Sheet{
		Title:   "کاربران",
		Headers: []string{"شناسه", "نام", "موبایل", "نقش‌ها", "وضعیت", "تاریخ عضویت"},
		Rows:    rows,
	})
}
