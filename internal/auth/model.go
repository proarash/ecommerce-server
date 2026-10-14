package auth

import (
	"time"

	"github.com/proarash/ecommerce-server/internal/user"
	"gorm.io/gorm"
)

const (
	DeviceBrowser = 1
	DeviceApp     = 2
)

type RefreshToken struct {
	gorm.Model
	SessionID  string     `json:"-" gorm:"uniqueIndex;not null"`
	TokenHash  string     `json:"-" gorm:"uniqueIndex;not null"`
	UserID     uint       `json:"user_id" gorm:"index;not null"`
	User       *user.User `json:"-" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	DeviceType int        `json:"device_type" gorm:"not null;default:1" enums:"1,2"`
	UserAgent  string     `json:"user_agent"`
	LastIP     string     `json:"last_ip"`
	ExpiresAt  time.Time  `json:"expires_at" gorm:"not null"`
}

type Device struct {
	Type      int
	UserAgent string
	IP        string
}
