package middleware

import (
	"net/http"
	"slices"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/types"
	"github.com/proarash/ecommerce-server/pkg/token"
)

const RoleAdmin = "admin"

func RequireRoles(roles ...string) gin.HandlerFunc {
	return func(c *gin.Context) {
		p := GetAuth(c)
		if p == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse{Error: "unauthorized"})
			return
		}
		if p.UserType == token.UserTypeStaff && (p.Role == RoleAdmin || slices.Contains(roles, p.Role)) {
			c.Next()
			return
		}
		c.AbortWithStatusJSON(http.StatusForbidden, types.ErrorResponse{Error: "forbidden"})
	}
}

func RequireCustomer() gin.HandlerFunc {
	return func(c *gin.Context) {
		p := GetAuth(c)
		if p == nil || p.UserType != token.UserTypeCustomer {
			c.AbortWithStatusJSON(http.StatusForbidden, types.ErrorResponse{Error: "customer access only"})
			return
		}
		c.Next()
	}
}
