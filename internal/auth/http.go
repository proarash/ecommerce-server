package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/types"
)

type AuthHandler interface {
	RegisterRoutes(router gin.IRouter)
	Login(ctx *gin.Context)
	Register(ctx *gin.Context)
}

type authHandler struct {
	repo   AuthRepo
	domain string
	secure bool
}

func NewAuthHandler(repo AuthRepo, domain string, secure bool) AuthHandler {
	return &authHandler{repo: repo, domain: domain, secure: secure}
}

func (h *authHandler) RegisterRoutes(router gin.IRouter) {
	g := router.Group("/auth")
	g.POST("/login", h.Login)
	g.POST("/register", h.Register)
}

func (h *authHandler) respond(c *gin.Context, status int, res TokenResponse, err error) {
	switch {
	case errors.Is(err, ErrInvalidCredentials):
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: err.Error()})
	case errors.Is(err, ErrInactive):
		c.JSON(http.StatusForbidden, types.ErrorResponse{Error: err.Error()})
	case errors.Is(err, ErrMobileTaken):
		c.JSON(http.StatusConflict, types.ErrorResponse{Error: err.Error()})
	case err != nil:
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: err.Error()})
	default:
		c.SetSameSite(http.SameSiteStrictMode)
		c.SetCookie("access_token", res.AccessToken, int(tokenTTL.Seconds()), "/", h.domain, h.secure, true)
		c.JSON(status, res)
	}
}

// Login godoc
// @Summary Login
// @Description Staff and customer login with mobile/password. user_type is optional; when omitted staff accounts are checked first.
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body LoginDto true "Credentials"
// @Success 200 {object} types.ApiResponse{data=TokenResponse}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 401 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/auth/login [post]
func (h *authHandler) Login(c *gin.Context) {
	var dto LoginDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	res, err := h.repo.Login(c.Request.Context(), dto)
	h.respond(c, http.StatusOK, res, err)
}

// Register godoc
// @Summary Customer registration
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body RegisterDto true "Registration data"
// @Success 201 {object} types.ApiResponse{data=TokenResponse}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 409 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /api/auth/register [post]
func (h *authHandler) Register(c *gin.Context) {
	var dto RegisterDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	res, err := h.repo.Register(c.Request.Context(), dto)
	h.respond(c, http.StatusCreated, res, err)
}
