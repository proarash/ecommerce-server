package auth

import (
	"github.com/gin-gonic/gin"
)

type AuthHandler interface {
	RegisterRoutes(router gin.IRouter)
	SignIn(ctx *gin.Context)
}

type authHandler struct {
	repo AuthRepo
}

func (h *authHandler) RegisterRoutes(router gin.IRouter) {
	g := router.Group("/auth")
	g.POST("/sign-in", h.SignIn)
}

func NewAuthHandler(repo AuthRepo) AuthHandler {
	return &authHandler{repo: repo}
}

// SignIn implements [AuthHandler].
func (h *authHandler) SignIn(c *gin.Context) {
	var authDto SignInDto
	if err := c.ShouldBindJSON(&authDto); err != nil {
		c.JSON(400, gin.H{
			"msg":    "Bad request",
			"fields": &authDto,
		})
		return
	}
	result := h.repo.SignIn(authDto, c)
	if result == "" {
		c.JSON(401, "error while generating auth token")
		return
	}
	c.SetCookie("access_token", result, 3600, "/", "localhost", false, true)
	c.JSON(200, result)
}
