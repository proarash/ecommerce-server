package wallet

import (
	"context"
	"errors"

	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

var (
	ErrLocked              = errors.New("this wallet cannot be modified")
	ErrInsufficientBalance = errors.New("insufficient wallet balance")
	ErrInvalidAmount       = errors.New("amount must be greater than zero")
)

type Store interface {
	Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error
	Backfill(ctx context.Context) error
	FindByID(ctx context.Context, id uint) (Wallet, error)
	FindByOwner(ctx context.Context, ownerType string, ownerID uint) (Wallet, error)
	List(ctx context.Context, q ListQuery) ([]Wallet, int64, error)
	ListTransactions(ctx context.Context, walletID uint, q TransactionQuery) ([]WalletTransaction, int64, error)
	Adjust(ctx context.Context, id uint, e Entry) (Wallet, WalletTransaction, error)
	ApplyTx(tx *gorm.DB, ownerType string, ownerID uint, e Entry) (Wallet, WalletTransaction, error)
}

type store struct {
	db *gorm.DB
}

func NewStore(db *gorm.DB) Store {
	return &store{db: db}
}

func (s *store) Transaction(ctx context.Context, fn func(tx *gorm.DB) error) error {
	return s.db.WithContext(ctx).Transaction(fn)
}

func (s *store) Backfill(ctx context.Context) error {
	db := s.db.WithContext(ctx)
	if err := db.Exec(`INSERT INTO wallets (created_at, updated_at, owner_id, owner_type, amount, locked)
		SELECT NOW(), NOW(), id, ?, 0, false FROM users WHERE deleted_at IS NULL
		ON CONFLICT (owner_type, owner_id) DO NOTHING`, OwnerUser).Error; err != nil {
		return err
	}
	if err := db.Exec(`INSERT INTO wallets (created_at, updated_at, owner_id, owner_type, amount, locked)
		SELECT NOW(), NOW(), id, ?, 0, role = 'admin' FROM staff_users WHERE deleted_at IS NULL
		ON CONFLICT (owner_type, owner_id) DO NOTHING`, OwnerStaff).Error; err != nil {
		return err
	}
	return db.Exec(`UPDATE wallets SET locked = true WHERE owner_type = ? AND locked = false
		AND owner_id IN (SELECT id FROM staff_users WHERE role = 'admin')`, OwnerStaff).Error
}

func (s *store) FindByID(ctx context.Context, id uint) (Wallet, error) {
	return gorm.G[Wallet](s.db).Where("id = ?", id).First(ctx)
}

func (s *store) FindByOwner(ctx context.Context, ownerType string, ownerID uint) (Wallet, error) {
	return gorm.G[Wallet](s.db).Where("owner_type = ? AND owner_id = ?", ownerType, ownerID).First(ctx)
}

func (s *store) List(ctx context.Context, q ListQuery) ([]Wallet, int64, error) {
	tx := s.db.WithContext(ctx).Model(&Wallet{})
	if q.OwnerType != "" {
		tx = tx.Where("owner_type = ?", q.OwnerType)
	}
	if q.OwnerID != 0 {
		tx = tx.Where("owner_id = ?", q.OwnerID)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []Wallet
	err := tx.Order("id DESC").Offset(q.Offset()).Limit(q.Limit).Find(&items).Error
	return items, total, err
}

func (s *store) ListTransactions(ctx context.Context, walletID uint, q TransactionQuery) ([]WalletTransaction, int64, error) {
	tx := s.db.WithContext(ctx).Model(&WalletTransaction{}).Where("wallet_id = ?", walletID)
	if q.Type != "" {
		tx = tx.Where("type = ?", q.Type)
	}
	if q.Source != "" {
		tx = tx.Where("source = ?", q.Source)
	}
	var total int64
	if err := tx.Count(&total).Error; err != nil {
		return nil, 0, err
	}
	var items []WalletTransaction
	err := tx.Order("id DESC").Offset(q.Offset()).Limit(q.Limit).Find(&items).Error
	return items, total, err
}

func (s *store) Adjust(ctx context.Context, id uint, e Entry) (Wallet, WalletTransaction, error) {
	var w Wallet
	var t WalletTransaction
	err := s.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		var err error
		w, t, err = apply(tx, tx.Where("id = ?", id), e)
		return err
	})
	return w, t, err
}

func (s *store) ApplyTx(tx *gorm.DB, ownerType string, ownerID uint, e Entry) (Wallet, WalletTransaction, error) {
	if err := CreateFor(tx, ownerType, ownerID, false); err != nil {
		return Wallet{}, WalletTransaction{}, err
	}
	return apply(tx, tx.Where("owner_type = ? AND owner_id = ?", ownerType, ownerID), e)
}

func apply(tx *gorm.DB, scope *gorm.DB, e Entry) (Wallet, WalletTransaction, error) {
	if e.Amount <= 0 {
		return Wallet{}, WalletTransaction{}, ErrInvalidAmount
	}
	var w Wallet
	if err := scope.Clauses(clause.Locking{Strength: "UPDATE"}).First(&w).Error; err != nil {
		return Wallet{}, WalletTransaction{}, err
	}
	if w.Locked {
		return Wallet{}, WalletTransaction{}, ErrLocked
	}
	before := w.Amount
	after := before + e.Amount
	if e.Type == TxDebit {
		if before < e.Amount {
			return Wallet{}, WalletTransaction{}, ErrInsufficientBalance
		}
		after = before - e.Amount
	}
	if err := tx.Model(&w).Update("amount", after).Error; err != nil {
		return Wallet{}, WalletTransaction{}, err
	}
	w.Amount = after
	t := WalletTransaction{WalletID: w.ID, Type: e.Type, Source: e.Source, Amount: e.Amount, BalanceBefore: before, BalanceAfter: after, OrderID: e.OrderID, TrackID: e.TrackID, PerformedByID: e.PerformedByID, Description: e.Description}
	if err := tx.Create(&t).Error; err != nil {
		return Wallet{}, WalletTransaction{}, err
	}
	return w, t, nil
}
