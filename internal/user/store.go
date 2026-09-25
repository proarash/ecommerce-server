package user

import (
	"context"

	"gorm.io/gorm"
)

type Store interface {
	Create(ctx context.Context, u *User) error
	FindByID(ctx context.Context, id uint) (User, error)
	FindByMobile(ctx context.Context, mobile string) (User, error)
	Update(ctx context.Context, id uint, fields map[string]any) error
}

type store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) Store {
	return &store{db: db}
}

func (s *store) Create(ctx context.Context, u *User) error {
	return gorm.G[User](s.db).Create(ctx, u)
}

func (s *store) FindByID(ctx context.Context, id uint) (User, error) {
	return gorm.G[User](s.db).Where("id = ?", id).First(ctx)
}

func (s *store) FindByMobile(ctx context.Context, mobile string) (User, error) {
	return gorm.G[User](s.db).Where("mobile = ?", mobile).First(ctx)
}

func (s *store) Update(ctx context.Context, id uint, fields map[string]any) error {
	res := s.db.WithContext(ctx).Model(&User{}).Where("id = ?", id).Updates(fields)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
