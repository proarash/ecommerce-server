package staff

import (
	"github.com/proarash/ecommerce-server/internal/media"
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
	Name            string       `json:"name"`
	Mobile          string       `json:"mobile" gorm:"uniqueIndex;not null"`
	Password        string       `json:"-"`
	Role            string       `json:"role" gorm:"index;not null"`
	AvatarMediaID   *uint        `json:"avatar_media_id"`
	AvatarMedia     *media.Media `json:"avatar_media,omitempty"`
	DefaultAvatarID *int         `json:"default_avatar_id" enums:"1,2,3,4,5"`
	Status          bool         `json:"status" gorm:"default:true"`
}
