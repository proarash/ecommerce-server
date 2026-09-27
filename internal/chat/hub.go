package chat

import (
	"context"
	"errors"

	"encoding/json"
	"gorm.io/gorm"
	"sync"
)

type Hub struct {
	mu       sync.RWMutex
	rooms    map[uint]map[*Client]struct{}
	supports map[*Client]struct{}
}

func NewHub() *Hub {
	return &Hub{rooms: map[uint]map[*Client]struct{}{}, supports: map[*Client]struct{}{}}
}

func (h *Hub) register(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if c.isSupport {
		h.supports[c] = struct{}{}
		return
	}
	if h.rooms[c.roomID] == nil {
		h.rooms[c.roomID] = map[*Client]struct{}{}
	}
	h.rooms[c.roomID][c] = struct{}{}
}

func (h *Hub) unregister(c *Client) {
	h.mu.Lock()
	defer h.mu.Unlock()
	if c.isSupport {
		if _, ok := h.supports[c]; ok {
			delete(h.supports, c)
			close(c.send)
		}
		return
	}
	if set, ok := h.rooms[c.roomID]; ok {
		if _, ok := set[c]; ok {
			delete(set, c)
			close(c.send)
		}
		if len(set) == 0 {
			delete(h.rooms, c.roomID)
		}
	}
}

func (h *Hub) Publish(msg ChatMessage) {
	data, err := json.Marshal(msg)
	if err != nil {
		return
	}
	h.mu.RLock()
	defer h.mu.RUnlock()
	for c := range h.rooms[msg.RoomID] {
		c.trySend(data)
	}
	for c := range h.supports {
		c.trySend(data)
	}
}

type Service struct {
	db            *gorm.DB
	hub           *Hub
	OnRoomCreated func(ctx context.Context, room ChatRoom)
}

func NewService(db *gorm.DB, hub *Hub) *Service {
	return &Service{db: db, hub: hub}
}

func (s *Service) EnsureRoom(ctx context.Context, userID uint) (ChatRoom, bool, error) {
	room, err := gorm.G[ChatRoom](s.db).Where("user_id = ? AND status <> ?", userID, RoomClosed).Order("id DESC").First(ctx)
	if err == nil {
		return room, false, nil
	}
	if !errors.Is(err, gorm.ErrRecordNotFound) {
		return ChatRoom{}, false, err
	}
	room = ChatRoom{UserID: userID, Status: RoomOpen}
	if err := gorm.G[ChatRoom](s.db).Create(ctx, &room); err != nil {
		return ChatRoom{}, false, err
	}
	return room, true, nil
}

func (s *Service) Post(ctx context.Context, msg *ChatMessage) error {
	if msg.Type == "" {
		msg.Type = TypeText
	}
	if err := gorm.G[ChatMessage](s.db).Create(ctx, msg); err != nil {
		return err
	}
	s.hub.Publish(*msg)
	return nil
}

func (s *Service) UserMessage(ctx context.Context, roomID, userID uint, text string) (ChatMessage, error) {
	msg := ChatMessage{RoomID: roomID, SenderRole: SenderUser, SenderID: &userID, Message: text}
	return msg, s.Post(ctx, &msg)
}

func (s *Service) SupportReply(ctx context.Context, roomID, supportID uint, text string) (ChatMessage, error) {
	room, err := gorm.G[ChatRoom](s.db).Where("id = ?", roomID).First(ctx)
	if err != nil {
		return ChatMessage{}, err
	}
	if room.Status == RoomClosed {
		return ChatMessage{}, errors.New("room is closed")
	}
	if room.SupportID == nil || room.Status == RoomOpen {
		if err := s.db.WithContext(ctx).Model(&ChatRoom{}).Where("id = ?", roomID).Updates(map[string]any{"support_id": supportID, "status": RoomInProgress}).Error; err != nil {
			return ChatMessage{}, err
		}
	}
	msg := ChatMessage{RoomID: roomID, SenderRole: SenderSupport, SenderID: &supportID, Message: text}
	return msg, s.Post(ctx, &msg)
}

func (s *Service) BotMessage(ctx context.Context, userID uint, text, msgType string, payload *string) (ChatMessage, error) {
	room, _, err := s.EnsureRoom(ctx, userID)
	if err != nil {
		return ChatMessage{}, err
	}
	msg := ChatMessage{RoomID: room.ID, SenderRole: SenderBot, Message: text, Type: msgType, Payload: payload}
	return msg, s.Post(ctx, &msg)
}

func (s *Service) UserMessages(ctx context.Context, userID uint, offset, limit int) ([]ChatMessage, int64, error) {
	q := s.db.WithContext(ctx).Model(&ChatMessage{}).Where("room_id IN (?)", s.db.WithContext(ctx).Model(&ChatRoom{}).Select("id").Where("user_id = ?", userID))
	return paginate(q, offset, limit)
}

func (s *Service) RoomMessages(ctx context.Context, roomID uint, offset, limit int) ([]ChatMessage, int64, error) {
	q := s.db.WithContext(ctx).Model(&ChatMessage{}).Where("room_id = ?", roomID)
	return paginate(q, offset, limit)
}

func paginate(q *gorm.DB, offset, limit int) ([]ChatMessage, int64, error) {
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []ChatMessage
	err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}

func (s *Service) Rooms(ctx context.Context, status string, offset, limit int) ([]ChatRoom, int64, error) {
	q := s.db.WithContext(ctx).Model(&ChatRoom{})
	if status != "" {
		q = q.Where("status = ?", status)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []ChatRoom
	err := q.Order("updated_at DESC").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}

func (s *Service) UpdateRoomStatus(ctx context.Context, roomID uint, status string) error {
	res := s.db.WithContext(ctx).Model(&ChatRoom{}).Where("id = ?", roomID).Update("status", status)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
