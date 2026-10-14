package product

import (
	"context"
	"errors"
	"fmt"
	"strings"

	"github.com/proarash/ecommerce-server/internal/media"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrCategoryCycle       = errors.New("category cannot be its own ancestor")
	ErrCategoryHasProducts = errors.New("category still has products; move or delete them first")
)

type Store interface {
	CreateCategory(ctx context.Context, c *Category) error
	ListCategories(ctx context.Context) ([]Category, error)
	FindCategory(ctx context.Context, id uint) (Category, error)
	UpdateCategory(ctx context.Context, id uint, fields map[string]any) error
	DeleteCategory(ctx context.Context, id uint) error
	CreateProduct(ctx context.Context, p *Product, attributes []AttributeValuesInput) error
	ListProducts(ctx context.Context, q ProductQuery, onlyActive bool) ([]Product, int64, error)
	FindProduct(ctx context.Context, id uint) (Product, error)
	UpdateProduct(ctx context.Context, id uint, fields map[string]any, attributes *[]AttributeValuesInput) error
	DeleteProduct(ctx context.Context, id uint) error
	CreateAttribute(ctx context.Context, a *Attribute) error
	ListAttributes(ctx context.Context, q AttributeQuery) ([]Attribute, int64, error)
	FindAttribute(ctx context.Context, id uint) (Attribute, error)
	UpdateAttribute(ctx context.Context, id uint, fields map[string]any) error
	DeleteAttribute(ctx context.Context, id uint) error
	AttachAttributes(ctx context.Context, productID uint, attributes []AttributeValuesInput) error
	DetachAttribute(ctx context.Context, productID, attributeID uint, value string) error
	AttachMediaIDs(ctx context.Context, productID uint, mediaIDs []uint) error
	DetachMedia(ctx context.Context, productID, mediaID uint) error
	SetCategoryMedia(ctx context.Context, categoryID uint, mediaID *uint) error
}

var ErrAttributeNotFound = errors.New("one or more attributes do not exist")

var ErrMediaNotFound = errors.New("one or more media do not exist")

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
		var products int64
		if err := tx.Model(&Product{}).Where("category_id = ?", id).Count(&products).Error; err != nil {
			return err
		}
		if products > 0 {
			return ErrCategoryHasProducts
		}
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
	unique := map[uint]struct{}{}
	for _, id := range ids {
		unique[id] = struct{}{}
	}
	if len(items) != len(unique) {
		return nil, ErrMediaNotFound
	}
	return items, nil
}

func buildAttributeValues(tx *gorm.DB, productID uint, inputs []AttributeValuesInput) ([]ProductAttributeValue, error) {
	rows := []ProductAttributeValue{}
	if len(inputs) == 0 {
		return rows, nil
	}
	ids := map[uint]struct{}{}
	seen := map[string]struct{}{}
	for _, in := range inputs {
		ids[in.AttributeID] = struct{}{}
		for _, raw := range in.Values {
			value := strings.TrimSpace(raw)
			key := fmt.Sprintf("%d|%s", in.AttributeID, value)
			if value == "" {
				continue
			}
			if _, dup := seen[key]; dup {
				continue
			}
			seen[key] = struct{}{}
			rows = append(rows, ProductAttributeValue{ProductID: productID, AttributeID: in.AttributeID, Value: value})
		}
	}
	list := make([]uint, 0, len(ids))
	for id := range ids {
		list = append(list, id)
	}
	var found int64
	if err := tx.Model(&Attribute{}).Where("id IN ?", list).Count(&found).Error; err != nil {
		return nil, err
	}
	if int(found) != len(list) {
		return nil, ErrAttributeNotFound
	}
	return rows, nil
}

func insertAttributeValues(tx *gorm.DB, rows []ProductAttributeValue) error {
	if len(rows) == 0 {
		return nil
	}
	return tx.Clauses(clause.OnConflict{DoNothing: true}).Create(&rows).Error
}

func preloadAttributeValues(tx *gorm.DB) *gorm.DB {
	return tx.Preload("AttributeValues.Attribute")
}

func (s *store) CreateProduct(ctx context.Context, p *Product, attributes []AttributeValuesInput) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Omit("AttributeValues").Create(p).Error; err != nil {
			return err
		}
		rows, err := buildAttributeValues(tx, p.ID, attributes)
		if err != nil {
			return err
		}
		return insertAttributeValues(tx, rows)
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
	err := preloadAttributeValues(tx).Preload("Media").Preload("Category").Order("id DESC").Offset((q.Page - 1) * q.Limit).Limit(q.Limit).Find(&items).Error
	return items, total, err
}

func (s *store) FindProduct(ctx context.Context, id uint) (Product, error) {
	var p Product
	err := preloadAttributeValues(s.db.WithContext(ctx)).Preload("Media").Preload("Category").First(&p, id).Error
	return p, err
}

func (s *store) UpdateProduct(ctx context.Context, id uint, fields map[string]any, attributes *[]AttributeValuesInput) error {
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
		if attributes != nil {
			rows, err := buildAttributeValues(tx, p.ID, *attributes)
			if err != nil {
				return err
			}
			if err := tx.Where("product_id = ?", p.ID).Delete(&ProductAttributeValue{}).Error; err != nil {
				return err
			}
			if err := insertAttributeValues(tx, rows); err != nil {
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
		tx = tx.Where("key ILIKE ? OR name ILIKE ?", like, like)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Attribute
	err := tx.Order("name, id").Offset((q.Page - 1) * q.Limit).Limit(q.Limit).Find(&items).Error
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
		if err := tx.Where("attribute_id = ?", id).Delete(&ProductAttributeValue{}).Error; err != nil {
			return err
		}
		res := tx.Delete(&Attribute{}, id)
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (s *store) AttachAttributes(ctx context.Context, productID uint, attributes []AttributeValuesInput) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p Product
		if err := tx.First(&p, productID).Error; err != nil {
			return err
		}
		rows, err := buildAttributeValues(tx, p.ID, attributes)
		if err != nil {
			return err
		}
		return insertAttributeValues(tx, rows)
	})
}

func (s *store) DetachAttribute(ctx context.Context, productID, attributeID uint, value string) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p Product
		if err := tx.First(&p, productID).Error; err != nil {
			return err
		}
		q := tx.Where("product_id = ? AND attribute_id = ?", productID, attributeID)
		if value = strings.TrimSpace(value); value != "" {
			q = q.Where("value = ?", value)
		}
		res := q.Delete(&ProductAttributeValue{})
		if res.Error != nil {
			return res.Error
		}
		if res.RowsAffected == 0 {
			return gorm.ErrRecordNotFound
		}
		return nil
	})
}

func (s *store) AttachMediaIDs(ctx context.Context, productID uint, mediaIDs []uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p Product
		if err := tx.First(&p, productID).Error; err != nil {
			return err
		}
		items, err := loadMedia(tx, mediaIDs)
		if err != nil {
			return err
		}
		return tx.Model(&p).Association("Media").Append(items)
	})
}

func (s *store) DetachMedia(ctx context.Context, productID, mediaID uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p Product
		if err := tx.First(&p, productID).Error; err != nil {
			return err
		}
		var m media.Media
		if err := tx.First(&m, mediaID).Error; err != nil {
			return err
		}
		return tx.Model(&p).Association("Media").Delete(&m)
	})
}

func (s *store) SetCategoryMedia(ctx context.Context, categoryID uint, mediaID *uint) error {
	res := s.db.WithContext(ctx).Model(&Category{}).Where("id = ?", categoryID).Update("media_id", mediaID)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}
