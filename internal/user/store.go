package user

import (
	"context"

	"gorm.io/gorm"
)

type Store interface {
	Create(ctx context.Context, u *User) error
	FindByID(ctx context.Context, id uint) (User, error)
	FindByMobile(ctx context.Context, mobile string) (User, error)
	List(ctx context.Context, q ListQuery) ([]User, int64, error)
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
	return gorm.G[User](s.db).Preload("AvatarMedia", nil).Where("id = ?", id).First(ctx)
}

func (s *store) FindByMobile(ctx context.Context, mobile string) (User, error) {
	return gorm.G[User](s.db).Where("mobile = ?", mobile).First(ctx)
}

func (s *store) List(ctx context.Context, q ListQuery) ([]User, int64, error) {
	tx := s.db.WithContext(ctx).Model(&User{})
	if q.Role != "" {
		tx = tx.Where("roles @> ?::jsonb", RoleFilter(q.Role))
	}
	if q.Staff {
		tx = tx.Where("jsonb_array_length(roles) > 1")
	}
	if q.Search != "" {
		like := "%" + q.Search + "%"
		tx = tx.Where("mobile ILIKE ? OR name ILIKE ?", like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []User
	err := tx.Preload("AvatarMedia").Order("id").Offset((q.Page - 1) * q.Limit).Limit(q.Limit).Find(&items).Error
	return items, total, err
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
