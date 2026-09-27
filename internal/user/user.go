package user

import (
	"github.com/proarash/ecommerce-server/internal/wallet"
	"gorm.io/gorm"
)

type User struct {
	gorm.Model
	Name           *string        `json:"name"`
	Mobile         string         `json:"mobile" gorm:"uniqueIndex;not null"`
	Password       string         `json:"-"`
	TelegramChatID *int64         `json:"telegram_chat_id" gorm:"index"`
	Address        *string        `json:"address"`
	Latitude       *float64       `json:"latitude"`
	Longitude      *float64       `json:"longitude"`
	Status         bool           `json:"status" gorm:"default:true"`
	Wallet         *wallet.Wallet `json:"wallet,omitempty" gorm:"polymorphic:Owner;polymorphicValue:user"`
}

func (u *User) AfterCreate(tx *gorm.DB) error {
	return wallet.CreateFor(tx, wallet.OwnerUser, u.ID, false)
}
