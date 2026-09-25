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
	CreateProduct(ctx context.Context, p *Product, mediaIDs, attributeIDs []uint) error
	ListProducts(ctx context.Context, q ProductQuery, onlyActive bool) ([]Product, int64, error)
	FindProduct(ctx context.Context, id uint) (Product, error)
	UpdateProduct(ctx context.Context, id uint, fields map[string]any, mediaIDs, attributeIDs *[]uint) error
	DeleteProduct(ctx context.Context, id uint) error
	CreateAttribute(ctx context.Context, a *Attribute) error
	ListAttributes(ctx context.Context, q AttributeQuery) ([]Attribute, int64, error)
	FindAttribute(ctx context.Context, id uint) (Attribute, error)
	UpdateAttribute(ctx context.Context, id uint, fields map[string]any) error
	DeleteAttribute(ctx context.Context, id uint) error
	AttachAttributes(ctx context.Context, productID uint, attributeIDs []uint) error
	DetachAttribute(ctx context.Context, productID, attributeID uint) error
}

var ErrAttributeNotFound = errors.New("one or more attributes do not exist")

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

func loadAttributes(tx *gorm.DB, ids []uint) ([]Attribute, error) {
	var items []Attribute
	if len(ids) == 0 {
		return items, nil
	}
	if err := tx.Where("id IN ?", ids).Find(&items).Error; err != nil {
		return nil, err
	}
	unique := map[uint]struct{}{}
	for _, id := range ids {
		unique[id] = struct{}{}
	}
	if len(items) != len(unique) {
		return nil, ErrAttributeNotFound
	}
	return items, nil
}

func (s *store) CreateProduct(ctx context.Context, p *Product, mediaIDs, attributeIDs []uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		items, err := loadMedia(tx, mediaIDs)
		if err != nil {
			return err
		}
		p.Media = items
		if p.Attributes, err = loadAttributes(tx, attributeIDs); err != nil {
			return err
		}
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
	err := tx.Preload("Media").Preload("Attributes").Preload("Category").Order("id DESC").Offset((q.Page - 1) * q.Limit).Limit(q.Limit).Find(&items).Error
	return items, total, err
}

func (s *store) FindProduct(ctx context.Context, id uint) (Product, error) {
	return gorm.G[Product](s.db).Preload("Media", nil).Preload("Attributes", nil).Preload("Category", nil).Where("id = ?", id).First(ctx)
}

func (s *store) UpdateProduct(ctx context.Context, id uint, fields map[string]any, mediaIDs, attributeIDs *[]uint) error {
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
		if attributeIDs != nil {
			attrs, err := loadAttributes(tx, *attributeIDs)
			if err != nil {
				return err
			}
			if err := tx.Model(&p).Association("Attributes").Replace(attrs); err != nil {
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

func (s *store) CreateAttribute(ctx context.Context, a *Attribute) error {
	return gorm.G[Attribute](s.db).Create(ctx, a)
}

func (s *store) ListAttributes(ctx context.Context, q AttributeQuery) ([]Attribute, int64, error) {
	tx := s.db.WithContext(ctx).Model(&Attribute{})
	if q.Search != "" {
		like := "%" + q.Search + "%"
		tx = tx.Where("key ILIKE ? OR title ILIKE ? OR name ILIKE ?", like, like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Attribute
	err := tx.Order("key, id").Offset((q.Page - 1) * q.Limit).Limit(q.Limit).Find(&items).Error
	return items, total, err
}

func (s *store) FindAttribute(ctx context.Context, id uint) (Attribute, error) {
	return gorm.G[Attribute](s.db).Where("id = ?", id).First(ctx)
}

func (s *store) UpdateAttribute(ctx context.Context, id uint, fields map[string]any) error {
	res := s.db.WithContext(ctx).Model(&Attribute{}).Where("id = ?", id).Updates(fields)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *store) DeleteAttribute(ctx context.Context, id uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var a Attribute
		if err := tx.First(&a, id).Error; err != nil {
			return err
		}
		if err := tx.Model(&a).Association("Products").Clear(); err != nil {
			return err
		}
		return tx.Delete(&a).Error
	})
}

func (s *store) AttachAttributes(ctx context.Context, productID uint, attributeIDs []uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p Product
		if err := tx.First(&p, productID).Error; err != nil {
			return err
		}
		attrs, err := loadAttributes(tx, attributeIDs)
		if err != nil {
			return err
		}
		return tx.Model(&p).Association("Attributes").Append(attrs)
	})
}

func (s *store) DetachAttribute(ctx context.Context, productID, attributeID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p Product
		if err := tx.First(&p, productID).Error; err != nil {
			return err
		}
		var a Attribute
		if err := tx.First(&a, attributeID).Error; err != nil {
			return err
		}
		return tx.Model(&p).Association("Attributes").Delete(&a)
	})
}
