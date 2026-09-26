package staff

import (
	"github.com/proarash/ecommerce-server/internal/media"
	"github.com/proarash/ecommerce-server/internal/wallet"
	"gorm.io/gorm"
)

const (
	RoleAdmin       = "admin"
	RoleStorekeeper = "storekeeper"
	RoleAccountant  = "accountant"
	RoleMarketer    = "marketer"
	RoleSupport     = "support"
)

type StaffUser struct {
	gorm.Model
	Name            string         `json:"name"`
	Mobile          string         `json:"mobile" gorm:"uniqueIndex;not null"`
	Password        string         `json:"-"`
	Role            string         `json:"role" gorm:"index;not null"`
	AvatarMediaID   *uint          `json:"avatar_media_id"`
	AvatarMedia     *media.Media   `json:"avatar_media,omitempty"`
	DefaultAvatarID *int           `json:"default_avatar_id" enums:"1,2,3,4,5"`
	Status          bool           `json:"status" gorm:"default:true"`
	Wallet          *wallet.Wallet `json:"wallet,omitempty" gorm:"polymorphic:Owner;polymorphicValue:staff"`
}

func (s *StaffUser) AfterCreate(tx *gorm.DB) error {
	return wallet.CreateFor(tx, wallet.OwnerStaff, s.ID, s.Role == RoleAdmin)
}
