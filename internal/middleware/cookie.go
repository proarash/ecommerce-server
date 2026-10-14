package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/pkg/token"
)

const AccessTokenCookie = "x-access-token"

type CookieWriter struct {
	Domain string
	Secure bool
}

func (w CookieWriter) domain() string {
	if w.Domain == "localhost" {
		return ""
	}
	return w.Domain
}

func (w CookieWriter) Set(c *gin.Context, value string) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(AccessTokenCookie, value, int(token.RefreshTTL.Seconds()), "/", w.domain(), w.Secure, true)
}

func (w CookieWriter) Clear(c *gin.Context) {
	c.SetSameSite(http.SameSiteLaxMode)
	c.SetCookie(AccessTokenCookie, "", -1, "/", w.domain(), w.Secure, true)
}
