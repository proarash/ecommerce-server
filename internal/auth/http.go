package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/middleware"
	"github.com/proarash/ecommerce-server/internal/types"
	"github.com/proarash/ecommerce-server/pkg/token"
)

type AuthHandler interface {
	RegisterRoutes(router gin.IRouter, admin gin.IRouter)
	StaffLogin(ctx *gin.Context)
	UserLogin(ctx *gin.Context)
	Register(ctx *gin.Context)
	Logout(ctx *gin.Context)
	Impersonate(ctx *gin.Context)
}

type authHandler struct {
	repo    AuthRepo
	cookies middleware.CookieWriter
	secret  string
}

func NewAuthHandler(repo AuthRepo, cookies middleware.CookieWriter, secret string) AuthHandler {
	return &authHandler{repo: repo, cookies: cookies, secret: secret}
}

func (h *authHandler) RegisterRoutes(router gin.IRouter, admin gin.IRouter) {
	g := router.Group("/auth")
	g.POST("/staff/login", h.StaffLogin)
	g.POST("/user/login", h.UserLogin)
	g.POST("/register", h.Register)
	g.GET("/logout", h.Logout)
	admin.POST("/users/:id/impersonate", h.Impersonate)
}

func (h *authHandler) respond(c *gin.Context, status int, userType string, res TokenResponse, err error) {
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
		name, _ := middleware.AccessCookie(userType)
		h.cookies.Set(c, name, res.AccessToken)
		c.JSON(status, res)
	}
}

// StaffLogin godoc
// @Summary Staff login
// @Description Staff login with mobile/password. Sets the staff_access_token httpOnly cookie; the refresh token is stored server side and replaces any previous staff session.
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body LoginDto true "Credentials"
// @Success 200 {object} types.ApiResponse{data=TokenResponse}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 401 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /auth/staff/login [post]
func (h *authHandler) StaffLogin(c *gin.Context) {
	var dto LoginDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	res, err := h.repo.LoginStaff(c.Request.Context(), dto)
	h.respond(c, http.StatusOK, token.UserTypeStaff, res, err)
}

// UserLogin godoc
// @Summary Customer login
// @Description Customer login with mobile/password. Sets the user_access_token httpOnly cookie; the refresh token is stored server side and replaces any previous customer session.
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body LoginDto true "Credentials"
// @Success 200 {object} types.ApiResponse{data=TokenResponse}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 401 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /auth/user/login [post]
func (h *authHandler) UserLogin(c *gin.Context) {
	var dto LoginDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	res, err := h.repo.LoginCustomer(c.Request.Context(), dto)
	h.respond(c, http.StatusOK, token.UserTypeCustomer, res, err)
}

// Register godoc
// @Summary Customer registration
// @Description Registers a customer and sets the user_access_token httpOnly cookie.
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body RegisterDto true "Registration data"
// @Success 201 {object} types.ApiResponse{data=TokenResponse}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 409 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /auth/register [post]
func (h *authHandler) Register(c *gin.Context) {
	var dto RegisterDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	res, err := h.repo.Register(c.Request.Context(), dto)
	h.respond(c, http.StatusCreated, token.UserTypeCustomer, res, err)
}

// Logout godoc
// @Summary Logout
// @Description Deletes the server side refresh token and clears the access cookie for the given user_type, or for both sessions when user_type is omitted.
// @Tags Auth
// @Produce json
// @Param user_type query string false "Session to clear" Enums(staff, customer)
// @Success 200 {object} types.ApiResponse{data=types.MessageResponse}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /auth/logout [get]
func (h *authHandler) Logout(c *gin.Context) {
	userType := c.Query("user_type")
	targets := []string{token.UserTypeStaff, token.UserTypeCustomer}
	if userType != "" {
		if _, ok := middleware.AccessCookie(userType); !ok {
			c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "invalid user_type"})
			return
		}
		targets = []string{userType}
	}
	for _, t := range targets {
		name, _ := middleware.AccessCookie(t)
		if raw, err := c.Cookie(name); err == nil && raw != "" {
			if p, err := token.ParseExpiredJwt(raw, h.secret); err == nil && p.UserType == t {
				if err := h.repo.Revoke(c.Request.Context(), p); err != nil {
					c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: err.Error()})
					return
				}
			}
		}
		h.cookies.Clear(c, name)
	}
	c.JSON(http.StatusOK, types.MessageResponse{Message: "logged out"})
}

// Impersonate godoc
// @Summary Login as customer
// @Description Admin only. Issues a 2h customer access token for the given user without mobile/password and sets it as the user_access_token httpOnly cookie. No refresh token is stored, so the session ends when the access token expires. The token carries impersonator_id with the admin's ID.
// @Tags Admin
// @Produce json
// @Security BearerAuth
// @Param user_type query string false "Session to authenticate with; selects the staff or customer auth cookie" Enums(staff, customer)
// @Param id path int true "Customer user ID"
// @Success 200 {object} types.ApiResponse{data=TokenResponse}
// @Failure 400 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 401 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 403 {object} types.ApiResponse{data=types.ErrorResponse}
// @Failure 404 {object} types.ApiResponse{data=types.ErrorResponse}
// @Router /admin/users/{id}/impersonate [post]
func (h *authHandler) Impersonate(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	res, err := h.repo.Impersonate(c.Request.Context(), middleware.GetAuth(c).UserID, id)
	if err != nil {
		types.HandleError(c, err, "user not found")
		return
	}
	name, _ := middleware.AccessCookie(token.UserTypeCustomer)
	h.cookies.Set(c, name, res.AccessToken)
	c.JSON(http.StatusOK, res)
}
