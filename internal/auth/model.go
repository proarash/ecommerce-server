package auth

import (
	"time"

	"github.com/proarash/ecommerce-server/internal/staff"
	"github.com/proarash/ecommerce-server/internal/user"
	"gorm.io/gorm"
)

type RefreshToken struct {
	gorm.Model
	Token     string           `json:"-" gorm:"uniqueIndex;not null"`
	StaffID   *uint            `json:"staff_id" gorm:"uniqueIndex"`
	Staff     *staff.StaffUser `json:"-" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	UserID    *uint            `json:"user_id" gorm:"uniqueIndex"`
	User      *user.User       `json:"-" gorm:"constraint:OnUpdate:CASCADE,OnDelete:CASCADE;"`
	ExpiresAt time.Time        `json:"expires_at" gorm:"not null"`
}
