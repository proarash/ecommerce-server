package auth

import (
	"time"

	"github.com/proarash/ecommerce-server/internal/user"
)

type LoginDto struct {
	Mobile   string `json:"mobile" binding:"required,numeric,len=11" example:"09120000000"`
	Password string `json:"password" binding:"required,min=6,max=72" example:"secret123"`
}

type RegisterDto struct {
	Name     *string `json:"name" binding:"omitempty,min=2,max=100"`
	Mobile   string  `json:"mobile" binding:"required,numeric,len=11" example:"09120000000"`
	Password string  `json:"password" binding:"required,min=6,max=72" example:"secret123"`
}

type TokenResponse struct {
	UserID       uint       `json:"user_id"`
	Roles        []string   `json:"roles" enums:"user,admin,storekeeper,accountant,marketer,support"`
	DeviceType   int        `json:"device_type" enums:"1,2"`
	RefreshToken string     `json:"refresh_token,omitempty"`
	ExpiresAt    *time.Time `json:"expires_at,omitempty"`
}

type MeResponse struct {
	Roles          []string  `json:"roles" enums:"user,admin,storekeeper,accountant,marketer,support"`
	ImpersonatorID *uint     `json:"impersonator_id,omitempty"`
	Profile        user.User `json:"profile"`
}
