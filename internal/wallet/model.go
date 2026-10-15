package wallet

import (
	"gorm.io/gorm"
	"gorm.io/gorm/clause"
)

const (
	TxCredit = "credit"
	TxDebit  = "debit"

	SourceGateway  = "gateway"
	SourcePurchase = "purchase"
	SourceManual   = "manual"
	SourceRefund   = "refund"
)

type Wallet struct {
	gorm.Model
	OwnerID uint    `json:"owner_id" gorm:"not null;uniqueIndex"`
	Amount  float64 `json:"amount" gorm:"not null;default:0"`
	Locked  bool    `json:"locked" gorm:"not null;default:false"`
}

type WalletTransaction struct {
	gorm.Model
	WalletID      uint    `json:"wallet_id" gorm:"index;not null"`
	Type          string  `json:"type" gorm:"size:16;not null" enums:"credit,debit"`
	Source        string  `json:"source" gorm:"size:16;index;not null" enums:"gateway,purchase,manual,refund"`
	Amount        float64 `json:"amount" gorm:"not null"`
	BalanceBefore float64 `json:"balance_before"`
	BalanceAfter  float64 `json:"balance_after"`
	OrderID       *uint   `json:"order_id" gorm:"index"`
	TrackID       *int64  `json:"track_id" gorm:"index"`
	PerformedByID *uint   `json:"performed_by_id"`
	Description   string  `json:"description"`
}

func CreateFor(tx *gorm.DB, ownerID uint, locked bool) error {
	w := Wallet{OwnerID: ownerID, Locked: locked}
	return tx.Clauses(clause.OnConflict{Columns: []clause.Column{{Name: "owner_id"}}, DoNothing: true}).Create(&w).Error
}
