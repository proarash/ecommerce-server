package payment

import (
	"context"

	"gorm.io/gorm"
)

type Store interface {
	Create(ctx context.Context, t *PaymentTransaction) error
	FindByTrackID(ctx context.Context, trackID int64) (PaymentTransaction, error)
	FindUserTransaction(ctx context.Context, trackID int64, userID uint) (PaymentTransaction, error)
	Save(ctx context.Context, t *PaymentTransaction) error
}

type store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) Store {
	return &store{db: db}
}

func (s *store) Create(ctx context.Context, t *PaymentTransaction) error {
	return gorm.G[PaymentTransaction](s.db).Create(ctx, t)
}

func (s *store) FindByTrackID(ctx context.Context, trackID int64) (PaymentTransaction, error) {
	return gorm.G[PaymentTransaction](s.db).Where("track_id = ?", trackID).First(ctx)
}

func (s *store) FindUserTransaction(ctx context.Context, trackID int64, userID uint) (PaymentTransaction, error) {
	return gorm.G[PaymentTransaction](s.db).Where("track_id = ? AND user_id = ?", trackID, userID).First(ctx)
}

func (s *store) Save(ctx context.Context, t *PaymentTransaction) error {
	return s.db.WithContext(ctx).Save(t).Error
}
