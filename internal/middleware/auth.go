package middleware

import (
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/types"
	"github.com/proarash/ecommerce-server/pkg/token"
)

const authKey = "auth"

func extractToken(c *gin.Context) string {
	if h := c.GetHeader("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimPrefix(h, "Bearer ")
	}
	if strings.HasPrefix(c.Request.URL.Path, "/ws/") {
		if q := c.Query("token"); q != "" {
			return q
		}
	}
	if ck, err := c.Cookie("access_token"); err == nil {
		return ck
	}
	return ""
}

func Auth(secret string) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := extractToken(c)
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse{Error: "missing token"})
			return
		}
		payload, err := token.ParseJwt(raw, secret)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse{Error: "invalid token"})
			return
		}
		c.Set(authKey, payload)
		c.Set("user_id", payload.UserID)
		c.Set("role", payload.Role)
		c.Set("user_type", payload.UserType)
		c.Next()
	}
}

func GetAuth(c *gin.Context) *token.AuthPayload {
	v, ok := c.Get(authKey)
	if !ok {
		return nil
	}
	p, _ := v.(*token.AuthPayload)
	return p
}
