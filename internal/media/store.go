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
}

type store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) Store {
	return &store{db: db}
}

func targetTable(target int) string {
	if target == TargetCategory {
		return "categories"
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
	if target == TargetCategory {
		return s.db.WithContext(ctx).Table("categories").Where("id = ? AND deleted_at IS NULL", id).Update("media_id", items[0].ID).Error
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
