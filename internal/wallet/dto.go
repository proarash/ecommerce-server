package wallet

type Entry struct {
	Type          string
	Source        string
	Amount        float64
	OrderID       *uint
	TrackID       *int64
	PerformedByID *uint
	Description   string
}

type AdjustDto struct {
	Type        string  `json:"type" binding:"required,oneof=credit debit" enums:"credit,debit"`
	Amount      float64 `json:"amount" binding:"required,gt=0" example:"50000"`
	Description string  `json:"description" binding:"required,max=255"`
}

type ListQuery struct {
	Page      int    `form:"page,default=1" binding:"min=1"`
	Limit     int    `form:"limit,default=20" binding:"min=1,max=100"`
	OwnerType string `form:"owner_type" binding:"omitempty,oneof=user staff"`
	OwnerID   uint   `form:"owner_id"`
}

func (q ListQuery) Offset() int {
	return (q.Page - 1) * q.Limit
}

type TransactionQuery struct {
	Page   int    `form:"page,default=1" binding:"min=1"`
	Limit  int    `form:"limit,default=20" binding:"min=1,max=100"`
	Type   string `form:"type" binding:"omitempty,oneof=credit debit"`
	Source string `form:"source" binding:"omitempty,oneof=gateway purchase manual"`
}

func (q TransactionQuery) Offset() int {
	return (q.Page - 1) * q.Limit
}

type AdjustResponse struct {
	Wallet      Wallet            `json:"wallet"`
	Transaction WalletTransaction `json:"transaction"`
}
