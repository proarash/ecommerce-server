package inventory

import (
	"context"
	"errors"

	"github.com/proarash/ecommerce-server/internal/product"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var ErrInsufficientStock = errors.New("insufficient stock")

type Store interface {
	List(ctx context.Context, offset, limit int) ([]InventoryStock, int64, error)
	Inbound(ctx context.Context, dto InboundDto, staffID uint) (InventoryStock, error)
	Outbound(ctx context.Context, dto OutboundDto, staffID uint) (InventoryStock, error)
	Logs(ctx context.Context, productID uint, offset, limit int) ([]InventoryLog, int64, error)
}

type store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) Store {
	return &store{db: db}
}

func (s *store) List(ctx context.Context, offset, limit int) ([]InventoryStock, int64, error) {
	q := s.db.WithContext(ctx).Model(&InventoryStock{})
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []InventoryStock
	err := q.Preload("Product").Order("product_id").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}

func lockStock(tx *gorm.DB, productID uint) (InventoryStock, error) {
	if err := tx.First(&product.Product{}, productID).Error; err != nil {
		return InventoryStock{}, err
	}
	stock := InventoryStock{ProductID: productID}
	if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "product_id"}}, DoNothing: true}).Create(&stock).Error; err != nil {
		return InventoryStock{}, err
	}
	stock = InventoryStock{}
	err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("product_id = ?", productID).First(&stock).Error
	return stock, err
}

func (s *store) Inbound(ctx context.Context, dto InboundDto, staffID uint) (InventoryStock, error) {
	var stock InventoryStock
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		if stock, err = lockStock(tx, dto.ProductID); err != nil {
			return err
		}
		stock.Quantity += dto.Quantity
		if dto.WarehouseLocation != nil {
			stock.WarehouseLocation = dto.WarehouseLocation
		}
		if err := tx.Save(&stock).Error; err != nil {
			return err
		}
		return tx.Create(&InventoryLog{ProductID: dto.ProductID, Type: LogInbound, Quantity: dto.Quantity, Reason: dto.SupplierNote, CreatedBy: staffID}).Error
	})
	return stock, err
}

func (s *store) Outbound(ctx context.Context, dto OutboundDto, staffID uint) (InventoryStock, error) {
	var stock InventoryStock
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		stock, err = dispatch(tx, dto.ProductID, dto.Quantity, dto.Reason, staffID)
		return err
	})
	return stock, err
}

func dispatch(tx *gorm.DB, productID uint, quantity int, reason string, createdBy uint) (InventoryStock, error) {
	stock, err := lockStock(tx, productID)
	if err != nil {
		return InventoryStock{}, err
	}
	if stock.Quantity < quantity {
		return InventoryStock{}, ErrInsufficientStock
	}
	stock.Quantity -= quantity
	if err := tx.Save(&stock).Error; err != nil {
		return InventoryStock{}, err
	}
	err = tx.Create(&InventoryLog{ProductID: productID, Type: LogOutbound, Quantity: quantity, Reason: reason, CreatedBy: createdBy}).Error
	return stock, err
}

func DispatchTx(tx *gorm.DB, productID uint, quantity int, reason string) error {
	_, err := dispatch(tx, productID, quantity, reason, 0)
	return err
}

func (s *store) Logs(ctx context.Context, productID uint, offset, limit int) ([]InventoryLog, int64, error) {
	q := s.db.WithContext(ctx).Model(&InventoryLog{}).Where("product_id = ?", productID)
	var total int64
	if err := q.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []InventoryLog
	err := q.Order("id DESC").Offset(offset).Limit(limit).Find(&items).Error
	return items, total, err
}
