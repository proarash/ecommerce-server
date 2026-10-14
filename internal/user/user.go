package user

import (
	"github.com/proarash/ecommerce-server/internal/media"
	"github.com/proarash/ecommerce-server/internal/wallet"
	"gorm.io/gorm"
)

const (
	RoleUser        = "user"
	RoleAdmin       = "admin"
	RoleStorekeeper = "storekeeper"
	RoleAccountant  = "accountant"
	RoleMarketer    = "marketer"
	RoleSupport     = "support"
)

type User struct {
	gorm.Model
	Name            *string        `json:"name"`
	Mobile          string         `json:"mobile" gorm:"uniqueIndex;not null"`
	Password        string         `json:"-"`
	Roles           Roles          `json:"roles" gorm:"type:jsonb;not null;default:'[\"user\"]'" swaggertype:"array,string" enums:"user,admin,storekeeper,accountant,marketer,support"`
	AvatarMediaID   *uint          `json:"avatar_media_id"`
	AvatarMedia     *media.Media   `json:"avatar_media,omitempty"`
	DefaultAvatarID *int           `json:"default_avatar_id" enums:"1,2,3,4,5"`
	TelegramChatID  *int64         `json:"telegram_chat_id" gorm:"index"`
	Address         *string        `json:"address"`
	Latitude        *float64       `json:"latitude"`
	Longitude       *float64       `json:"longitude"`
	Status          bool           `json:"status" gorm:"default:true"`
	Wallet          *wallet.Wallet `json:"wallet,omitempty" gorm:"foreignKey:OwnerID"`
}

func (u *User) AfterCreate(tx *gorm.DB) error {
	return wallet.CreateFor(tx, u.ID, u.Roles.Has(RoleAdmin))
}
