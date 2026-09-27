package notification

import "gorm.io/gorm"

const (
	TypeOrder   = "order"
	TypePayment = "payment"
	TypeSystem  = "system"
	TypePromo   = "promo"

	ChannelInApp    = "in_app"
	ChannelTelegram = "telegram"
	ChannelBoth     = "both"
)

type Notification struct {
	gorm.Model
	UserID  *uint  `json:"user_id" gorm:"index"`
	Title   string `json:"title"`
	Message string `json:"message"`
	Type    string `json:"type" enums:"order,payment,system,promo"`
	IsRead  bool   `json:"is_read" gorm:"default:false"`
	Channel string `json:"channel" enums:"in_app,telegram,both"`
}

type NotificationRead struct {
	gorm.Model
	NotificationID uint `json:"notification_id" gorm:"uniqueIndex:idx_notification_reads_notification_user;not null"`
	UserID         uint `json:"user_id" gorm:"uniqueIndex:idx_notification_reads_notification_user;not null"`
}
