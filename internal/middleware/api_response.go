package middleware

import (
	"bytes"
	"encoding/json"
	"net/http"
	"strings"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/types"
)

type responseWriter struct {
	gin.ResponseWriter
	body *bytes.Buffer
}

func (w responseWriter) Write(b []byte) (int, error) {
	return w.body.Write(b)
}

func (w responseWriter) WriteString(s string) (int, error) {
	return w.body.WriteString(s)
}

var skipPrefixes = []string{"/ws", "/swagger", "/api/payment/callback"}

func ApiResponseMiddleware(c *gin.Context) {
	for _, p := range skipPrefixes {
		if strings.HasPrefix(c.Request.URL.Path, p) {
			c.Next()
			return
		}
	}

	originalResponse := c.Writer
	writer := &responseWriter{
		ResponseWriter: c.Writer,
		body:           bytes.NewBuffer(nil),
	}

	c.Writer = writer
	c.Next()

	c.Writer = originalResponse

	statusCode := c.Writer.Status()
	if statusCode >= 300 && statusCode < 400 {
		return
	}

	var data any
	raw := writer.body.Bytes()
	if len(raw) > 0 && json.Valid(raw) {
		data = json.RawMessage(raw)
	} else if len(raw) > 0 {
		data = string(raw)
	}

	c.JSON(statusCode, types.ApiResponse{
		Message: http.StatusText(statusCode),
		Data:    data,
	})
}

func Cors(c *gin.Context) {
	origin := c.GetHeader("Origin")
	if origin != "" {
		c.Header("Access-Control-Allow-Origin", origin)
		c.Header("Vary", "Origin")
		c.Header("Access-Control-Allow-Credentials", "true")
		c.Header("Access-Control-Allow-Headers", "Authorization, Content-Type")
		c.Header("Access-Control-Allow-Methods", "GET, POST, PATCH, PUT, DELETE, OPTIONS")
	}
	if c.Request.Method == http.MethodOptions {
		c.AbortWithStatus(http.StatusNoContent)
		return
	}
	c.Next()
}
