package auth

import "github.com/gin-gonic/gin"

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
func (h *authHandler) SignIn(ctx *gin.Context) {
	var authDto SignInDto
	if err := ctx.ShouldBindJSON(&authDto); err != nil {
		ctx.JSON(400, gin.H{
			"msg":    "Bad request",
			"fields": &authDto,
		})
		return
	}
	ctx.JSON(200, gin.H{
		"msg": h.repo.SignIn(authDto),
	})
}
