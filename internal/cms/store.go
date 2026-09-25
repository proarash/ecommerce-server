package cms

import (
	"context"

	"github.com/proarash/ecommerce-server/internal/media"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

type Store interface {
	ListBlogs(ctx context.Context, q BlogQuery) ([]BlogPost, int64, error)
	FindBlogBySlug(ctx context.Context, slug string) (BlogPost, error)
	FindBlog(ctx context.Context, id uint) (BlogPost, error)
	CreateBlog(ctx context.Context, b *BlogPost, mediaIDs []uint) error
	UpdateBlog(ctx context.Context, id uint, fields map[string]any, mediaIDs *[]uint) error
	DeleteBlog(ctx context.Context, id uint) error
	ActiveBanners(ctx context.Context) ([]Banner, error)
	FindBanner(ctx context.Context, id uint) (Banner, error)
	CreateBanner(ctx context.Context, b *Banner) error
	UpdateBanner(ctx context.Context, id uint, fields map[string]any) error
	DeleteBanner(ctx context.Context, id uint) error
	ListContents(ctx context.Context) ([]SiteContent, error)
	UpsertContents(ctx context.Context, items []SiteContentItemDto) error
}

type store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) Store {
	return &store{db: db}
}

func (s *store) ListBlogs(ctx context.Context, q BlogQuery) ([]BlogPost, int64, error) {
	tx := s.db.WithContext(ctx).Model(&BlogPost{})
	if q.Search != "" {
		tx = tx.Where("title ILIKE ?", "%"+q.Search+"%")
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []BlogPost
	err := tx.Preload("Media").Omit("content").Order("id DESC").Offset((q.Page - 1) * q.Limit).Limit(q.Limit).Find(&items).Error
	return items, total, err
}

func (s *store) FindBlogBySlug(ctx context.Context, slug string) (BlogPost, error) {
	return gorm.G[BlogPost](s.db).Preload("Media", nil).Where("slug = ?", slug).First(ctx)
}

func (s *store) FindBlog(ctx context.Context, id uint) (BlogPost, error) {
	return gorm.G[BlogPost](s.db).Preload("Media", nil).Where("id = ?", id).First(ctx)
}

func loadMedia(tx *gorm.DB, ids []uint) ([]media.Media, error) {
	var items []media.Media
	if len(ids) == 0 {
		return items, nil
	}
	err := tx.Where("id IN ?", ids).Find(&items).Error
	return items, err
}

func (s *store) CreateBlog(ctx context.Context, b *BlogPost, mediaIDs []uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		items, err := loadMedia(tx, mediaIDs)
		if err != nil {
			return err
		}
		b.Media = items
		return tx.Create(b).Error
	})
}

func (s *store) UpdateBlog(ctx context.Context, id uint, fields map[string]any, mediaIDs *[]uint) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var b BlogPost
		if err := tx.First(&b, id).Error; err != nil {
			return err
		}
		if len(fields) > 0 {
			if err := tx.Model(&b).Updates(fields).Error; err != nil {
				return err
			}
		}
		if mediaIDs != nil {
			items, err := loadMedia(tx, *mediaIDs)
			if err != nil {
				return err
			}
			return tx.Model(&b).Association("Media").Replace(items)
		}
		return nil
	})
}

func deleteByID[T any](ctx context.Context, db *gorm.DB, id uint) error {
	n, err := gorm.G[T](db).Where("id = ?", id).Delete(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *store) DeleteBlog(ctx context.Context, id uint) error {
	return deleteByID[BlogPost](ctx, s.db, id)
}

func (s *store) ActiveBanners(ctx context.Context) ([]Banner, error) {
	return gorm.G[Banner](s.db).Preload("Media", nil).Where("is_active = ?", true).Order("display_order, id").Find(ctx)
}

func (s *store) FindBanner(ctx context.Context, id uint) (Banner, error) {
	return gorm.G[Banner](s.db).Preload("Media", nil).Where("id = ?", id).First(ctx)
}

func (s *store) CreateBanner(ctx context.Context, b *Banner) error {
	return s.db.WithContext(ctx).Create(b).Error
}

func (s *store) UpdateBanner(ctx context.Context, id uint, fields map[string]any) error {
	res := s.db.WithContext(ctx).Model(&Banner{}).Where("id = ?", id).Updates(fields)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *store) DeleteBanner(ctx context.Context, id uint) error {
	return deleteByID[Banner](ctx, s.db, id)
}

func (s *store) ListContents(ctx context.Context) ([]SiteContent, error) {
	return gorm.G[SiteContent](s.db).Preload("Media", nil).Order("key").Find(ctx)
}

func (s *store) UpsertContents(ctx context.Context, items []SiteContentItemDto) error {
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		for _, it := range items {
			row := SiteContent{Key: it.Key, Value: it.Value, MediaID: it.MediaID}
			cols := []string{"value", "media_id", "updated_at"}
			if it.Description != nil {
				row.Description = *it.Description
				cols = append(cols, "description")
			}
			if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "key"}}, DoUpdates: clause.AssignmentColumns(cols)}).Create(&row).Error; err != nil {
				return err
			}
		}
		return nil
	})
}
