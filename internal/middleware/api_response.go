package middleware

import (
	"bytes"
	"encoding/json"

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

func ApiResponseMiddleware(c *gin.Context) {
	originalResponse := c.Writer
	writer := &responseWriter{
		ResponseWriter: c.Writer,
		body:           bytes.NewBuffer(nil),
	}

	c.Writer = writer
	c.Next()

	c.Writer = originalResponse

	statusCode := c.Writer.Status()

	c.JSON(statusCode, types.ApiResponse{
		Message: "OK",
		Data:    json.RawMessage(writer.body.Bytes()),
	})

}
