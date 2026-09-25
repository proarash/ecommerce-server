package media

import (
	"context"

	"gorm.io/gorm"
)

type Store interface {
	Create(ctx context.Context, m *Media) error
	FindByID(ctx context.Context, id uint) (Media, error)
	FindByIDs(ctx context.Context, ids []uint) ([]Media, error)
}

type store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) Store {
	return &store{db: db}
}

func (s *store) Create(ctx context.Context, m *Media) error {
	return gorm.G[Media](s.db).Create(ctx, m)
}

func (s *store) FindByID(ctx context.Context, id uint) (Media, error) {
	return gorm.G[Media](s.db).Where("id = ?", id).First(ctx)
}

func (s *store) FindByIDs(ctx context.Context, ids []uint) ([]Media, error) {
	if len(ids) == 0 {
		return []Media{}, nil
	}
	return gorm.G[Media](s.db).Where("id IN ?", ids).Find(ctx)
}
