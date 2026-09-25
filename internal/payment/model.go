package payment

import (
	"time"

	"gorm.io/gorm"
)

const StatusPending = -1

type PaymentTransaction struct {
	gorm.Model
	OrderID     uint       `json:"order_id" gorm:"index;not null"`
	UserID      uint       `json:"user_id" gorm:"index;not null"`
	Amount      int64      `json:"amount"`
	TrackID     int64      `json:"track_id" gorm:"uniqueIndex;not null"`
	Status      int        `json:"status"`
	Result      int        `json:"result"`
	RefNumber   *int64     `json:"ref_number"`
	CardNumber  *string    `json:"card_number"`
	Description string     `json:"description"`
	PaidAt      *time.Time `json:"paid_at"`
}
