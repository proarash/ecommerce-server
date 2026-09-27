package middleware

import (
	"context"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/types"
	"github.com/proarash/ecommerce-server/pkg/token"
)

const authKey = "auth"

type Refresher interface {
	Refresh(ctx context.Context, expired *token.AuthPayload) (*token.AuthPayload, string, error)
}

func matches(p *token.AuthPayload, userType string) bool {
	return p != nil && (userType == "" || p.UserType == userType)
}

func fromCookie(c *gin.Context, secret, userType string, w CookieWriter, r Refresher) *token.AuthPayload {
	name, _ := AccessCookie(userType)
	raw, err := c.Cookie(name)
	if err != nil || raw == "" {
		return nil
	}
	if p, err := token.ParseJwt(raw, secret); err == nil {
		if matches(p, userType) {
			return p
		}
		return nil
	}
	expired, err := token.ParseExpiredJwt(raw, secret)
	if err != nil || !matches(expired, userType) || expired.ID == "" {
		return nil
	}
	p, access, err := r.Refresh(c.Request.Context(), expired)
	if err != nil {
		w.Clear(c, name)
		return nil
	}
	w.Set(c, name, access)
	return p
}

func Auth(secret string, w CookieWriter, r Refresher) gin.HandlerFunc {
	return func(c *gin.Context) {
		userType := c.Query("user_type")
		candidates := userTypes
		if userType != "" {
			if _, ok := AccessCookie(userType); !ok {
				c.AbortWithStatusJSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid user_type"})
				return
			}
			candidates = []string{userType}
		}
		var payload *token.AuthPayload
		if h := c.GetHeader("Authorization"); strings.HasPrefix(h, "Bearer ") {
			if p, err := token.ParseJwt(strings.TrimPrefix(h, "Bearer "), secret); err == nil && matches(p, userType) {
				payload = p
			}
		} else {
			for _, t := range candidates {
				if payload = fromCookie(c, secret, t, w, r); payload != nil {
					break
				}
			}
		}
		if payload == nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse{Error: "unauthorized"})
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
