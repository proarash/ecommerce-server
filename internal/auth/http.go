package auth

import (
	"errors"
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/proarash/ecommerce-server/internal/middleware"
	"github.com/proarash/ecommerce-server/internal/types"
	"github.com/proarash/ecommerce-server/internal/user"
	"github.com/proarash/ecommerce-server/pkg/token"
)

type AuthHandler interface {
	RegisterRoutes(router gin.IRouter, protected gin.IRouter, admin gin.IRouter)
	Login(ctx *gin.Context)
	Register(ctx *gin.Context)
	Logout(ctx *gin.Context)
	Me(ctx *gin.Context)
	UpdateMe(ctx *gin.Context)
	Impersonate(ctx *gin.Context)
}

type authHandler struct {
	repo    AuthRepo
	users   *user.Service
	secret  string
	cookies middleware.CookieWriter
}

func NewAuthHandler(repo AuthRepo, users *user.Service, secret string, cookies middleware.CookieWriter) AuthHandler {
	return &authHandler{repo: repo, users: users, secret: secret, cookies: cookies}
}

func (h *authHandler) RegisterRoutes(router gin.IRouter, protected gin.IRouter, admin gin.IRouter) {
	g := router.Group("/auth")
	g.POST("/login", h.Login)
	g.POST("/register", h.Register)
	g.POST("/logout", h.Logout)
	protected.GET("/me", h.Me)
	protected.PATCH("/me", h.UpdateMe)
	admin.POST("/users/:id/impersonate", h.Impersonate)
}

func deviceFrom(c *gin.Context) (Device, bool) {
	d := Device{Type: DeviceBrowser, UserAgent: c.Request.UserAgent(), IP: c.ClientIP()}
	switch c.GetHeader("X-Device-Type") {
	case "", "1":
	case "2":
		d.Type = DeviceApp
	default:
		c.JSON(http.StatusBadRequest, types.ErrorResponse{Error: "X-Device-Type must be 1 (browser) or 2 (native app)"})
		return Device{}, false
	}
	if len(d.UserAgent) > 512 {
		d.UserAgent = d.UserAgent[:512]
	}
	return d, true
}

func (h *authHandler) respond(c *gin.Context, status int, issued Issued, err error) {
	switch {
	case errors.Is(err, ErrInvalidCredentials), errors.Is(err, ErrSessionExpired):
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: err.Error()})
	case errors.Is(err, ErrInactive):
		c.JSON(http.StatusForbidden, types.ErrorResponse{Error: err.Error()})
	case errors.Is(err, ErrMobileTaken):
		c.JSON(http.StatusConflict, types.ErrorResponse{Error: err.Error()})
	case err != nil:
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: err.Error()})
	default:
		if issued.AccessToken != "" {
			h.cookies.Set(c, issued.AccessToken)
		}
		c.JSON(status, issued.Response)
	}
}

// Login godoc
// @Summary Login
// @Description Login with mobile/password for every user; the user's roles are read from the database and returned in the body. Every login creates a new session, so each device stays logged in independently; the session stores the device type, user agent and last IP.
// @Description X-Device-Type 1 (browser, default): a 24h access token is set as the httpOnly x-access-token cookie (SameSite=Lax, Secure in production); the browser sends it automatically. When it expires the server renews it from the stored session (90 days) and sets the new cookie on that same request, so the client never handles the token.
// @Description X-Device-Type 2 (native app): refresh_token (a 90 day JWT) is returned in the body and no cookie is set. Send it as the X-Access-Token request header (or Authorization: Bearer) on every request.
// @Tags Auth
// @Accept json
// @Produce json
// @Param X-Device-Type header int false "1 = browser (default), 2 = native app" Enums(1, 2)
// @Param body body LoginDto true "Credentials"
// @Success 200 {object} types.ApiResponse{data=TokenResponse}
// @Header 200 {string} Set-Cookie "x-access-token (httpOnly, browser only)"
// @Router /auth/login [post]
func (h *authHandler) Login(c *gin.Context) {
	device, ok := deviceFrom(c)
	if !ok {
		return
	}
	var dto LoginDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	issued, err := h.repo.Login(c.Request.Context(), dto, device)
	h.respond(c, http.StatusOK, issued, err)
}

// Register godoc
// @Summary Register
// @Description Registers a user with the user role and logs them in. Tokens are returned exactly like POST /auth/login, based on X-Device-Type.
// @Tags Auth
// @Accept json
// @Produce json
// @Param X-Device-Type header int false "1 = browser (default), 2 = native app" Enums(1, 2)
// @Param body body RegisterDto true "Registration data"
// @Success 201 {object} types.ApiResponse{data=TokenResponse}
// @Header 201 {string} Set-Cookie "x-access-token (httpOnly, browser only)"
// @Router /auth/register [post]
func (h *authHandler) Register(c *gin.Context) {
	device, ok := deviceFrom(c)
	if !ok {
		return
	}
	var dto RegisterDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	issued, err := h.repo.Register(c.Request.Context(), dto, device)
	h.respond(c, http.StatusCreated, issued, err)
}

// Logout godoc
// @Summary Logout
// @Description Ends only the current session: the session of the x-access-token cookie (browser, expired tokens accepted) or of the X-Access-Token request header (app) is deleted and the cookie is cleared. Other devices stay logged in.
// @Tags Auth
// @Produce json
// @Success 200 {object} types.ApiResponse{data=types.MessageResponse}
// @Router /auth/logout [post]
func (h *authHandler) Logout(c *gin.Context) {
	p, err := token.ParseExpiredJwt(middleware.RequestToken(c), h.secret)
	h.cookies.Clear(c)
	if err != nil {
		c.JSON(http.StatusUnauthorized, types.ErrorResponse{Error: "invalid token"})
		return
	}
	if err := h.repo.Revoke(c.Request.Context(), p); err != nil {
		c.JSON(http.StatusInternalServerError, types.ErrorResponse{Error: err.Error()})
		return
	}
	c.JSON(http.StatusOK, types.MessageResponse{Message: "logged out"})
}

func (h *authHandler) me(c *gin.Context, p *token.AuthPayload) {
	u, err := h.users.Profile(c.Request.Context(), p.UserID)
	if err != nil {
		types.HandleError(c, err, "account not found")
		return
	}
	c.JSON(http.StatusOK, MeResponse{Roles: u.Roles, ImpersonatorID: p.ImpersonatorID, Profile: u})
}

// Me godoc
// @Summary Get current account
// @Description Returns the logged in user (every account, including admin and staff roles, is a user). roles lists every role the user has (always includes user; admin has all roles).
// @Tags Auth
// @Produce json
// @Success 200 {object} types.ApiResponse{data=MeResponse}
// @Router /me [get]
func (h *authHandler) Me(c *gin.Context) {
	h.me(c, middleware.GetAuth(c))
}

// UpdateMe godoc
// @Summary Update current account
// @Description Updates the logged in user's own profile: name, mobile, avatar_media_id, default_avatar_id (preset frontend avatar 1..5), telegram_chat_id, address, latitude and longitude. Role and status can only be changed by an admin.
// @Tags Auth
// @Accept json
// @Produce json
// @Param body body user.UpdateProfileDto true "Profile fields"
// @Success 200 {object} types.ApiResponse{data=MeResponse}
// @Router /me [patch]
func (h *authHandler) UpdateMe(c *gin.Context) {
	var dto user.UpdateProfileDto
	if err := c.ShouldBindJSON(&dto); err != nil {
		types.BadRequest(c, err)
		return
	}
	p := middleware.GetAuth(c)
	if _, err := h.users.UpdateProfile(c.Request.Context(), p.UserID, dto); err != nil {
		types.HandleError(c, err, "account not found")
		return
	}
	h.me(c, p)
}

// Impersonate godoc
// @Summary Login as user
// @Description Admin only. Returns a 24h access token for the given user as the httpOnly x-access-token cookie (replacing the admin session cookie in this browser), without mobile/password. No session is stored, so it is not renewed and ends when it expires. The token carries impersonator_id with the admin's ID.
// @Tags Admin
// @Produce json
// @Param id path int true "User ID"
// @Success 200 {object} types.ApiResponse{data=TokenResponse}
// @Header 200 {string} Set-Cookie "x-access-token for the impersonated user (httpOnly)"
// @Router /admin/users/{id}/impersonate [post]
func (h *authHandler) Impersonate(c *gin.Context) {
	id, ok := types.ParamID(c, "id")
	if !ok {
		return
	}
	issued, err := h.repo.Impersonate(c.Request.Context(), middleware.GetAuth(c).UserID, id)
	if err != nil {
		types.HandleError(c, err, "user not found")
		return
	}
	h.cookies.Set(c, issued.AccessToken)
	c.JSON(http.StatusOK, issued.Response)
}
