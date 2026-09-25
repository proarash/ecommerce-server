package product

import (
	"context"
	"errors"

	"github.com/proarash/ecommerce-server/internal/media"
	"gorm.io/gorm"
)

var ErrCategoryCycle = errors.New("category cannot be its own ancestor")

type Store interface {
	CreateCategory(ctx context.Context, c *Category) error
	ListCategories(ctx context.Context) ([]Category, error)
	FindCategory(ctx context.Context, id uint) (Category, error)
	UpdateCategory(ctx context.Context, id uint, fields map[string]any) error
	DeleteCategory(ctx context.Context, id uint) error
	CreateProduct(ctx context.Context, p *Product, mediaIDs []uint) error
	ListProducts(ctx context.Context, q ProductQuery, onlyActive bool) ([]Product, int64, error)
	FindProduct(ctx context.Context, id uint) (Product, error)
	UpdateProduct(ctx context.Context, id uint, fields map[string]any, mediaIDs *[]uint) error
	DeleteProduct(ctx context.Context, id uint) error
}

type store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) Store {
	return &store{db: db}
}

func (s *store) CreateCategory(ctx context.Context, c *Category) error {
	return gorm.G[Category](s.db).Create(ctx, c)
}

func (s *store) ListCategories(ctx context.Context) ([]Category, error) {
	return gorm.G[Category](s.db).Preload("Media", nil).Order("id").Find(ctx)
}

func (s *store) FindCategory(ctx context.Context, id uint) (Category, error) {
	return gorm.G[Category](s.db).Preload("Media", nil).Preload("Children", nil).Where("id = ?", id).First(ctx)
}

func (s *store) ensureNoCycle(ctx context.Context, id uint, parentID uint) error {
	current := parentID
	for current != 0 {
		if current == id {
			return ErrCategoryCycle
		}
		c, err := gorm.G[Category](s.db).Where("id = ?", current).First(ctx)
		if err != nil {
			return err
		}
		if c.ParentID == nil {
			return nil
		}
		current = *c.ParentID
	}
	return nil
}

func (s *store) UpdateCategory(ctx context.Context, id uint, fields map[string]any) error {
	if pid, ok := fields["parent_id"].(uint); ok {
		if err := s.ensureNoCycle(ctx, id, pid); err != nil {
			return err
		}
	}
	res := s.db.WithContext(ctx).Model(&Category{}).Where("id = ?", id).Updates(fields)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *store) DeleteCategory(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Model(&Category{}).Where("parent_id = ?", id).Update("parent_id", nil).Error; err != nil {
			return err
		}
		res := tx.Delete(&Category{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func loadMedia(tx *gorm.DB, ids []uint) ([]media.Media, error) {
	var items []media.Media
	if len(ids) == 0 {
		return items, nil
	}
	if err := tx.Where("id IN ?", ids).Find(&items).Error; err != nil {
		return nil, err
	}
	return items, nil
}

func (s *store) CreateProduct(ctx context.Context, p *Product, mediaIDs []uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		items, err := loadMedia(tx, mediaIDs)
		if err != nil {
			return err
		}
		p.Media = items
		return tx.Create(p).Error
	})
}

func (s *store) ListProducts(ctx context.Context, q ProductQuery, onlyActive bool) ([]Product, int64, error) {
	tx := s.db.WithContext(ctx).Model(&Product{})
	if onlyActive {
		tx = tx.Where("is_active = ?", true)
	}
	if q.CategoryID != 0 {
		tx = tx.Where("category_id = ?", q.CategoryID)
	}
	if q.Search != "" {
		tx = tx.Where("title ILIKE ?", "%"+q.Search+"%")
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Product
	err := tx.Preload("Media").Preload("Category").Order("id DESC").Offset((q.Page - 1) * q.Limit).Limit(q.Limit).Find(&items).Error
	return items, total, err
}

func (s *store) FindProduct(ctx context.Context, id uint) (Product, error) {
	return gorm.G[Product](s.db).Preload("Media", nil).Preload("Category", nil).Where("id = ?", id).First(ctx)
}

func (s *store) UpdateProduct(ctx context.Context, id uint, fields map[string]any, mediaIDs *[]uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p Product
		if err := tx.First(&p, id).Error; err != nil {
			return err
		}
		if len(fields) > 0 {
			if err := tx.Model(&p).Updates(fields).Error; err != nil {
				return err
			}
		}
		if mediaIDs != nil {
			items, err := loadMedia(tx, *mediaIDs)
			if err != nil {
				return err
			}
			if err := tx.Model(&p).Association("Media").Replace(items); err != nil {
				return err
			}
		}
		return nil
	})
}

func (s *store) DeleteProduct(ctx context.Context, id uint) error {
	n, err := gorm.G[Product](s.db).Where("id = ?", id).Delete(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
