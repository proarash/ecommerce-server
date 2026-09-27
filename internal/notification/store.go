package notification

import (
	"context"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Store interface {
	Create(ctx context.Context, n *Notification) error
	FindByID(ctx context.Context, id uint) (Notification, error)
	List(ctx context.Context, userID *uint, offset, limit int) ([]Notification, int64, error)
	ListForUser(ctx context.Context, userID uint, offset, limit int) ([]Notification, int64, error)
	Update(ctx context.Context, id uint, fields map[string]any) error
	Delete(ctx context.Context, id uint) error
	MarkRead(ctx context.Context, id, userID uint) error
}

type store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) Store {
	return &store{db: db}
}

func (s *store) Create(ctx context.Context, n *Notification) error {
	return gorm.G[Notification](s.db).Create(ctx, n)
}

func (s *store) FindByID(ctx context.Context, id uint) (Notification, error) {
	return gorm.G[Notification](s.db).Where("id = ?", id).First(ctx)
}

func (s *store) page(q *gorm.DB, offset, limit int) ([]Notification, int64, error) {
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Notification
	err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}

func (s *store) List(ctx context.Context, userID *uint, offset, limit int) ([]Notification, int64, error) {
	q := s.db.WithContext(ctx).Model(&Notification{})
	if userID != nil {
		q = q.Where("user_id = ?", *userID)
	}
	return s.page(q, offset, limit)
}

func (s *store) ListForUser(ctx context.Context, userID uint, offset, limit int) ([]Notification, int64, error) {
	q := s.db.WithContext(ctx).Model(&Notification{}).Where("(user_id = ? OR user_id IS NULL) AND channel <> ?", userID, ChannelTelegram)
	items, total, err := s.page(q, offset, limit)
	if err != nil {
		return nil, 0, err
	}
	var broadcastIDs []uint
	for _, n := range items {
		if n.UserID == nil {
			broadcastIDs = append(broadcastIDs, n.ID)
		}
	}
	if len(broadcastIDs) == 0 {
		return items, total, nil
	}
	var readIDs []uint
	if err := s.db.WithContext(ctx).Model(&NotificationRead{}).Where("user_id = ? AND notification_id IN ?", userID, broadcastIDs).Pluck("notification_id", &readIDs).Error; err != nil {
		return nil, 0, err
	}
	read := map[uint]bool{}
	for _, id := range readIDs {
		read[id] = true
	}
	for i := range items {
		if items[i].UserID == nil {
			items[i].IsRead = read[items[i].ID]
		}
	}
	return items, total, nil
}

func (s *store) Update(ctx context.Context, id uint, fields map[string]any) error {
	res := s.db.WithContext(ctx).Model(&Notification{}).Where("id = ?", id).Updates(fields)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *store) Delete(ctx context.Context, id uint) error {
	n, err := gorm.G[Notification](s.db).Where("id = ?", id).Delete(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *store) MarkRead(ctx context.Context, id, userID uint) error {
	n, err := gorm.G[Notification](s.db).Where("id = ? AND (user_id = ? OR user_id IS NULL) AND channel <> ?", id, userID, ChannelTelegram).First(ctx)
	if err != nil {
		return err
	}
	if n.UserID != nil {
		return s.db.WithContext(ctx).Model(&Notification{}).Where("id = ?", id).Update("is_read", true).Error
	}
	return s.db.WithContext(ctx).Clauses(clause.OnConflict{DoNothing: true}).Create(&NotificationRead{NotificationID: id, UserID: userID}).Error
}
