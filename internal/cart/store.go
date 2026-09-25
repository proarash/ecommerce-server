package cart

import (
	"context"
	"errors"

	"github.com/proarash/ecommerce-server/internal/finance"
	"github.com/proarash/ecommerce-server/internal/product"
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrProductUnavailable = errors.New("product is not available")
	ErrEmptyCart          = errors.New("cart is empty")
)

type Store interface {
	Get(ctx context.Context, userID uint) (Cart, error)
	AddItem(ctx context.Context, userID uint, dto AddItemDto) error
	UpdateItem(ctx context.Context, userID, itemID uint, quantity int) error
	RemoveItem(ctx context.Context, userID, itemID uint) error
	Checkout(ctx context.Context, userID uint) (finance.Order, error)
}

type store struct {
	db      *gorm.DB
	finance finance.Store
}

func NewStore(db *gorm.DB, financeStore finance.Store) Store {
	return &store{db: db, finance: financeStore}
}

func ensureCart(tx *gorm.DB, userID uint) (Cart, error) {
	c := Cart{UserID: userID}
	if err := tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "user_id"}}, DoNothing: true}).Create(&c).Error; err != nil {
		return Cart{}, err
	}
	c = Cart{}
	err := tx.Where("user_id = ?", userID).First(&c).Error
	return c, err
}

func (s *store) Get(ctx context.Context, userID uint) (Cart, error) {
	c, err := ensureCart(s.db.WithContext(ctx), userID)
	if err != nil {
		return Cart{}, err
	}
	err = s.db.WithContext(ctx).Preload("Items", func(db *gorm.DB) *gorm.DB { return db.Order("id") }).Preload("Items.Product").Preload("Items.Product.Media").First(&c, c.ID).Error
	return c, err
}

func (s *store) AddItem(ctx context.Context, userID uint, dto AddItemDto) error {
	qty := dto.Quantity
	if qty == 0 {
		qty = 1
	}
	return s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var p product.Product
		if err := tx.First(&p, dto.ProductID).Error; err != nil {
			return err
		}
		if !p.IsActive {
			return ErrProductUnavailable
		}
		c, err := ensureCart(tx, userID)
		if err != nil {
			return err
		}
		var item CartItem
		err = tx.Where("cart_id = ? AND product_id = ?", c.ID, dto.ProductID).First(&item).Error
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return tx.Create(&CartItem{CartID: c.ID, ProductID: dto.ProductID, Quantity: qty}).Error
		}
		if err != nil {
			return err
		}
		return tx.Model(&item).Update("quantity", item.Quantity+qty).Error
	})
}

func (s *store) ownedItem(tx *gorm.DB, userID, itemID uint) *gorm.DB {
	return tx.Model(&CartItem{}).Where("id = ? AND cart_id IN (?)", itemID, tx.Session(&gorm.Session{NewDB: true}).Model(&Cart{}).Select("id").Where("user_id = ?", userID))
}

func (s *store) UpdateItem(ctx context.Context, userID, itemID uint, quantity int) error {
	res := s.ownedItem(s.db.WithContext(ctx), userID, itemID).Update("quantity", quantity)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *store) RemoveItem(ctx context.Context, userID, itemID uint) error {
	res := s.ownedItem(s.db.WithContext(ctx), userID, itemID).Unscoped().Delete(&CartItem{})
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *store) Checkout(ctx context.Context, userID uint) (finance.Order, error) {
	var order finance.Order
	var pre finance.PreInvoice
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var c Cart
		if err := tx.Clauses(clause.Locking{Strength: "UPDATE"}).Where("user_id = ?", userID).First(&c).Error; err != nil {
			if errors.Is(err, gorm.ErrRecordNotFound) {
				return ErrEmptyCart
			}
			return err
		}
		var items []CartItem
		if err := tx.Preload("Product").Where("cart_id = ?", c.ID).Find(&items).Error; err != nil {
			return err
		}
		lines := make([]finance.OrderLine, 0, len(items))
		for _, it := range items {
			if it.Product == nil || !it.Product.IsActive {
				return ErrProductUnavailable
			}
			lines = append(lines, finance.OrderLine{ProductID: it.ProductID, Quantity: it.Quantity, UnitPrice: it.Product.Price})
		}
		if len(lines) == 0 {
			return ErrEmptyCart
		}
		var err error
		if order, pre, err = s.finance.CreateOrderTx(tx, userID, lines); err != nil {
			return err
		}
		return tx.Unscoped().Where("cart_id = ?", c.ID).Delete(&CartItem{}).Error
	})
	if err != nil {
		return finance.Order{}, err
	}
	s.finance.NotifyOrderPlaced(ctx, order, pre)
	return s.finance.FindOrder(ctx, order.ID)
}
