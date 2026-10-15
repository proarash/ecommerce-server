package notification

import (
	"context"
	"fmt"
	"html"
	"log"

	"github.com/proarash/ecommerce-server/internal/user"
	"gorm.io/gorm"
)

const (
	MsgInvoice        = "invoice"
	MsgPreInvoice     = "preinvoice"
	MsgPaymentSuccess = "payment_success"
	MsgPaymentFailed  = "payment_failed"
	MsgSupport        = "support"
	MsgOrderReviewed  = "order_reviewed"
)

type AutomatedMessagePayload struct {
	UserID         uint
	TelegramChatID *int64
	Title          string
	Message        string
	Type           string
	ExtraData      map[string]interface{}
}

type Service struct {
	db       *gorm.DB
	store    Store
	telegram *TelegramClient
}

func NewService(db *gorm.DB, store Store, telegram *TelegramClient) *Service {
	return &Service{db: db, store: store, telegram: telegram}
}

func notificationType(t string) string {
	switch t {
	case MsgInvoice, MsgPreInvoice, MsgOrderReviewed:
		return TypeOrder
	case MsgPaymentSuccess, MsgPaymentFailed:
		return TypePayment
	default:
		return TypeSystem
	}
}

func formatTelegram(title, message string) string {
	return fmt.Sprintf("<b>%s</b>\n%s", html.EscapeString(title), html.EscapeString(message))
}

func (s *Service) SendAutomatedMessage(ctx context.Context, payload AutomatedMessagePayload) error {
	if payload.Type == MsgSupport {
		return s.telegram.SendAdmin(ctx, formatTelegram(payload.Title, payload.Message))
	}

	chatID := payload.TelegramChatID
	if chatID == nil {
		if u, err := gorm.G[user.User](s.db).Where("id = ?", payload.UserID).First(ctx); err == nil {
			chatID = u.TelegramChatID
		}
	}
	channel := ChannelInApp
	if chatID != nil && s.telegram.Enabled() {
		channel = ChannelBoth
	}

	uid := payload.UserID
	if err := s.store.Create(ctx, &Notification{UserID: &uid, Title: payload.Title, Message: payload.Message, Type: notificationType(payload.Type), Channel: channel}); err != nil {
		return err
	}

	if channel == ChannelBoth {
		text := formatTelegram(payload.Title, payload.Message)
		go func() {
			if err := s.telegram.SendMessage(context.Background(), *chatID, text); err != nil {
				log.Println("notification:", err)
			}
		}()
	}
	return nil
}

func (s *Service) Dispatch(ctx context.Context, n *Notification) error {
	if err := s.store.Create(ctx, n); err != nil {
		return err
	}
	if n.Channel == ChannelInApp || !s.telegram.Enabled() {
		return nil
	}
	text := formatTelegram(n.Title, n.Message)
	var chatIDs []int64
	q := s.db.WithContext(ctx).Model(&user.User{}).Where("telegram_chat_id IS NOT NULL")
	if n.UserID != nil {
		q = q.Where("id = ?", *n.UserID)
	}
	if err := q.Pluck("telegram_chat_id", &chatIDs).Error; err != nil {
		return err
	}
	go func() {
		for _, id := range chatIDs {
			if err := s.telegram.SendMessage(context.Background(), id, text); err != nil {
				log.Println("notification:", err)
			}
		}
	}()
	return nil
}
