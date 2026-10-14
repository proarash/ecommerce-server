package middleware

import (
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/types"
)

const RoleAdmin = "admin"

func RequireRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := GetAuth(c)
		if p == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse{Error: "unauthorized"})
			return
		}
		if p.HasRole(RoleAdmin) || slices.ContainsFunc(roles, p.HasRole) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, types.ErrorResponse{Error: "forbidden"})
	}
}
