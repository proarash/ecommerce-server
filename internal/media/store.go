package media

import (
	"context"

	"gorm.io/gorm"
)

type Store interface {
	Create(ctx context.Context, m *Media) error
	Update(ctx context.Context, m *Media) error
	Delete(ctx context.Context, id uint) error
	FindByID(ctx context.Context, id uint) (Media, error)
	FindByIDs(ctx context.Context, ids []uint) ([]Media, error)
	TargetExists(ctx context.Context, target int, id uint) error
	AssignToTarget(ctx context.Context, target int, id uint, items []Media) error
	List(ctx context.Context, q ListQuery) ([]Media, int64, error)
}

func (s *store) List(ctx context.Context, q ListQuery) ([]Media, int64, error) {
	tx := s.db.WithContext(ctx).Model(&Media{})
	if q.ID != 0 {
		tx = tx.Where("id = ?", q.ID)
	}
	if q.Type != "" {
		tx = tx.Where("mime_type = ?", q.Type)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Media
	err := tx.Order("id DESC").Offset((q.Page - 1) * q.Limit).Limit(q.Limit).Find(&items).Error
	return items, total, err
}

type store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) Store {
	return &store{db: db}
}

func targetTable(target int) string {
	switch target {
	case TargetCategory:
		return "categories"
	case TargetBanner:
		return "banners"
	}
	return "products"
}

func (s *store) Create(ctx context.Context, m *Media) error {
	return gorm.G[Media](s.db).Create(ctx, m)
}

func (s *store) Update(ctx context.Context, m *Media) error {
	return s.db.WithContext(ctx).Save(m).Error
}

func (s *store) Delete(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Exec("DELETE FROM product_media WHERE media_id = ?", id).Error; err != nil {
			return err
		}
		if err := tx.Table("categories").Where("media_id = ?", id).Update("media_id", nil).Error; err != nil {
			return err
		}
		if err := tx.Table("banners").Where("media_id = ?", id).Update("media_id", nil).Error; err != nil {
			return err
		}
		res := tx.Delete(&Media{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
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

func (s *store) TargetExists(ctx context.Context, target int, id uint) error {
	var count int64
	if err := s.db.WithContext(ctx).Table(targetTable(target)).Where("id = ? AND deleted_at IS NULL", id).Count(&count).Error; err != nil {
		return err
	}
	if count == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *store) AssignToTarget(ctx context.Context, target int, id uint, items []Media) error {
	if target == TargetCategory || target == TargetBanner {
		return s.db.WithContext(ctx).Table(targetTable(target)).Where("id = ? AND deleted_at IS NULL", id).Update("media_id", items[0].ID).Error
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, m := range items {
			if err := tx.Exec("INSERT INTO product_media (product_id, media_id) VALUES (?, ?) ON CONFLICT DO NOTHING", id, m.ID).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
