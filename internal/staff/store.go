package staff

import (
	"context"

	"gorm.io/gorm"
)

type Store interface {
	Create(ctx context.Context, s *StaffUser) error
	FindByID(ctx context.Context, id uint) (StaffUser, error)
	FindByMobile(ctx context.Context, mobile string) (StaffUser, error)
	List(ctx context.Context, role string, offset, limit int) ([]StaffUser, int64, error)
	Update(ctx context.Context, id uint, fields map[string]any) error
	ExistsRole(ctx context.Context, role string) (bool, error)
}

type store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) Store {
	return &store{db: db}
}

func (s *store) Create(ctx context.Context, u *StaffUser) error {
	return gorm.G[StaffUser](s.db).Create(ctx, u)
}

func (s *store) FindByID(ctx context.Context, id uint) (StaffUser, error) {
	return gorm.G[StaffUser](s.db).Preload("AvatarMedia", nil).Where("id = ?", id).First(ctx)
}

func (s *store) FindByMobile(ctx context.Context, mobile string) (StaffUser, error) {
	return gorm.G[StaffUser](s.db).Where("mobile = ?", mobile).First(ctx)
}

func (s *store) List(ctx context.Context, role string, offset, limit int) ([]StaffUser, int64, error) {
	q := s.db.WithContext(ctx).Model(&StaffUser{})
	if role != "" {
		q = q.Where("role = ?", role)
	}
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []StaffUser
	err := q.Preload("AvatarMedia").Order("id").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}

func (s *store) Update(ctx context.Context, id uint, fields map[string]any) error {
	res := s.db.WithContext(ctx).Model(&StaffUser{}).Where("id = ?", id).Updates(fields)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *store) ExistsRole(ctx context.Context, role string) (bool, error) {
	var count int64
	err := s.db.WithContext(ctx).Model(&StaffUser{}).Where("role = ?", role).Count(&count).Error
	return count > 0, err
}
