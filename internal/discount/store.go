package discount

import (
	"context"
	"crypto/rand"
	"errors"
	"strings"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	codeAlphabet = "ABCDEFGHJKLMNPQRSTUVWXYZ23456789"
	codeLength   = 8
)

var (
	ErrInvalidCode   = errors.New("discount code is invalid or inactive")
	ErrExhausted     = errors.New("discount code usage limit reached")
	ErrNotOwner      = errors.New("discount code belongs to another user")
	ErrPercentRange  = errors.New("percent discount value must be between 0 and 100")
	ErrMaxPriceValue = errors.New("max_price is only allowed for percent discounts")
)

type Store interface {
	Create(ctx context.Context, d *Discount) error
	FindByID(ctx context.Context, id uint) (Discount, error)
	List(ctx context.Context, q ListQuery) ([]Discount, int64, error)
	Update(ctx context.Context, id uint, fields map[string]any) error
	Delete(ctx context.Context, id uint) error
	Check(ctx context.Context, code string, userID uint, total float64) (Discount, float64, error)
	RedeemTx(tx *gorm.DB, code string, userID uint, total float64) (Discount, float64, error)
}

type store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) Store {
	return &store{db: db}
}

func NormalizeCode(code string) string {
	return strings.ToUpper(strings.TrimSpace(code))
}

func GenerateCode() string {
	b := make([]byte, codeLength)
	rand.Read(b)
	for i := range b {
		b[i] = codeAlphabet[int(b[i])%len(codeAlphabet)]
	}
	return string(b)
}

func Validate(d Discount) error {
	if d.Type == TypePercent && d.Value > 100 {
		return ErrPercentRange
	}
	if d.Type == TypeValue && d.MaxPrice != nil {
		return ErrMaxPriceValue
	}
	return nil
}

func (s *store) Create(ctx context.Context, d *Discount) error {
	if d.Code != "" {
		return gorm.G[Discount](s.db).Create(ctx, d)
	}
	var err error
	for range 5 {
		d.Code = GenerateCode()
		if err = gorm.G[Discount](s.db).Create(ctx, d); !errors.Is(err, gorm.ErrDuplicatedKey) {
			return err
		}
		d.ID = 0
	}
	return err
}

func (s *store) FindByID(ctx context.Context, id uint) (Discount, error) {
	return gorm.G[Discount](s.db).Preload("Owner", nil).Where("id = ?", id).First(ctx)
}

func (s *store) List(ctx context.Context, q ListQuery) ([]Discount, int64, error) {
	tx := s.db.WithContext(ctx).Model(&Discount{})
	if q.Code != "" {
		tx = tx.Where("code LIKE ?", "%"+NormalizeCode(q.Code)+"%")
	}
	if q.Type != "" {
		tx = tx.Where("type = ?", q.Type)
	}
	if q.Status != nil {
		tx = tx.Where("status = ?", *q.Status)
	}
	if q.OwnerID != 0 {
		tx = tx.Where("owner_id = ?", q.OwnerID)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Discount
	err := tx.Preload("Owner").Order("id DESC").Offset(q.Offset()).Limit(q.Limit).Find(&items).Error
	return items, total, err
}

func (s *store) Update(ctx context.Context, id uint, fields map[string]any) error {
	res := s.db.WithContext(ctx).Model(&Discount{}).Where("id = ?", id).Updates(fields)
	if res.Error != nil {
		return res.Error
	}
	if res.RowsAffected == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func (s *store) Delete(ctx context.Context, id uint) error {
	n, err := gorm.G[Discount](s.db).Where("id = ?", id).Delete(ctx)
	if err != nil {
		return err
	}
	if n == 0 {
		return gorm.ErrRecordNotFound
	}
	return nil
}

func usable(d Discount, userID uint) error {
	if !d.Status {
		return ErrInvalidCode
	}
	if d.UsedCount >= d.UseCount {
		return ErrExhausted
	}
	if d.OwnerID != nil && *d.OwnerID != userID {
		return ErrNotOwner
	}
	return nil
}

func find(tx *gorm.DB, code string) (Discount, error) {
	var d Discount
	err := tx.Where("code = ?", NormalizeCode(code)).First(&d).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return Discount{}, ErrInvalidCode
	}
	return d, err
}

func (s *store) Check(ctx context.Context, code string, userID uint, total float64) (Discount, float64, error) {
	d, err := find(s.db.WithContext(ctx), code)
	if err != nil {
		return Discount{}, 0, err
	}
	if err := usable(d, userID); err != nil {
		return Discount{}, 0, err
	}
	return d, d.Calculate(total), nil
}

func (s *store) RedeemTx(tx *gorm.DB, code string, userID uint, total float64) (Discount, float64, error) {
	d, err := find(tx.Clauses(clause.Locking{Strength: "UPDATE"}), code)
	if err != nil {
		return Discount{}, 0, err
	}
	if err := usable(d, userID); err != nil {
		return Discount{}, 0, err
	}
	if err := tx.Model(&d).Update("used_count", gorm.Expr("used_count + 1")).Error; err != nil {
		return Discount{}, 0, err
	}
	d.UsedCount++
	return d, d.Calculate(total), nil
}
