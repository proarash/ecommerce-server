package middleware

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/pkg/token"
)

var accessCookies = map[string]string{
	token.UserTypeStaff:    token.StaffCookie,
	token.UserTypeCustomer: token.CustomerCookie,
}

var userTypes = []string{token.UserTypeStaff, token.UserTypeCustomer}

func AccessCookie(userType string) (string, bool) {
	name, ok := accessCookies[userType]
	return name, ok
}

type CookieWriter struct {
	Domain string
	Secure bool
}

func (w CookieWriter) Set(c *gin.Context, name, value string) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(name, value, int(token.RefreshTTL.Seconds()), "/", w.Domain, w.Secure, true)
}

func (w CookieWriter) Clear(c *gin.Context, name string) {
	c.SetSameSite(http.SameSiteStrictMode)
	c.SetCookie(name, "", -1, "/", w.Domain, w.Secure, true)
}
