package chat

import "gorm.io/gorm"

const (
	RoomOpen       = "open"
	RoomInProgress = "in_progress"
	RoomClosed     = "closed"

	SenderBot     = "bot"
	SenderSupport = "support"
	SenderUser    = "user"

	TypeText           = "text"
	TypeInvoice        = "invoice"
	TypePreInvoice     = "preinvoice"
	TypePaymentSuccess = "payment_success"
	TypePaymentFailed  = "payment_failed"
)

type ChatRoom struct {
	gorm.Model
	UserID    uint   `json:"user_id" gorm:"index;not null"`
	SupportID *uint  `json:"support_id" gorm:"index"`
	Status    string `json:"status" gorm:"default:open" enums:"open,in_progress,closed"`
}

type ChatMessage struct {
	gorm.Model
	RoomID     uint    `json:"room_id" gorm:"index;not null"`
	SenderRole string  `json:"sender_role" enums:"bot,support,user"`
	SenderID   *uint   `json:"sender_id"`
	Message    string  `json:"message"`
	Type       string  `json:"type" gorm:"default:text" enums:"text,invoice,preinvoice,payment_success,payment_failed"`
	Payload    *string `json:"payload"`
}
