package discount

import (
	"github.com/proarash/ecommerce-server/internal/user"
	"gorm.io/gorm"
)

const (
	TypePercent = "percent"
	TypeValue   = "value"
)

type Discount struct {
	gorm.Model
	Code        string     `json:"code" gorm:"size:32;uniqueIndex;not null" example:"SUMMER30"`
	Type        string     `json:"type" gorm:"size:16;not null" enums:"percent,value"`
	Value       float64    `json:"value" gorm:"not null"`
	MaxPrice    *float64   `json:"max_price"`
	UseCount    int        `json:"use_count" gorm:"not null"`
	UsedCount   int        `json:"used_count" gorm:"not null;default:0"`
	Status      bool       `json:"status" gorm:"not null;default:true"`
	OwnerID     *uint      `json:"owner_id" gorm:"index"`
	Owner       *user.User `json:"owner,omitempty"`
	CreatedByID uint       `json:"created_by_id"`
}

func (d Discount) Calculate(total float64) float64 {
	amount := d.Value
	if d.Type == TypePercent {
		amount = total * d.Value / 100
		if d.MaxPrice != nil && amount > *d.MaxPrice {
			amount = *d.MaxPrice
		}
	}
	return min(amount, total)
}
