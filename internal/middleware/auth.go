package middleware

import (
	"context"
	"errors"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/types"
	"github.com/proarash/ecommerce-server/pkg/token"
)

const (
	authKey           = "auth"
	AccessTokenHeader = "X-Access-Token"
)

var errUnauthorized = errors.New("invalid or expired token")

type Renewer interface {
	Renew(ctx context.Context, expired *token.AuthPayload, ip string) (*token.AuthPayload, string, error)
}

func RequestToken(c *gin.Context) string {
	if v, err := c.Cookie(AccessTokenCookie); err == nil && v != "" {
		return v
	}
	if h := c.GetHeader("Authorization"); strings.HasPrefix(h, "Bearer ") {
		return strings.TrimSpace(strings.TrimPrefix(h, "Bearer "))
	}
	return strings.TrimSpace(c.GetHeader(AccessTokenHeader))
}

func Auth(secret string, r Renewer, w CookieWriter) gin.HandlerFunc {
	return authenticate(secret, r, w, false)
}

func AuthWithQueryToken(secret string, r Renewer, w CookieWriter) gin.HandlerFunc {
	return authenticate(secret, r, w, true)
}

func resolve(c *gin.Context, raw, secret string, r Renewer, w CookieWriter) (*token.AuthPayload, error) {
	p, err := token.ParseJwt(raw, secret)
	if err == nil {
		return p, nil
	}
	if !token.IsExpired(err) {
		return nil, errUnauthorized
	}
	expired, err := token.ParseExpiredJwt(raw, secret)
	if err != nil || expired.Kind != token.KindAccess {
		return nil, errUnauthorized
	}
	p, access, err := r.Renew(c.Request.Context(), expired, c.ClientIP())
	if err != nil {
		w.Clear(c)
		return nil, err
	}
	w.Set(c, access)
	return p, nil
}

func authenticate(secret string, r Renewer, w CookieWriter, allowQuery bool) gin.HandlerFunc {
	return func(c *gin.Context) {
		raw := RequestToken(c)
		if raw == "" && allowQuery {
			raw = c.Query("token")
		}
		if raw == "" {
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse{Error: "not authenticated"})
			return
		}
		payload, err := resolve(c, raw, secret, r, w)
		if err != nil {
			c.AbortWithStatusJSON(http.StatusUnauthorized, types.ErrorResponse{Error: err.Error()})
			return
		}
		c.Set(authKey, payload)
		c.Set("user_id", payload.UserID)
		c.Set("roles", payload.Roles)
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
