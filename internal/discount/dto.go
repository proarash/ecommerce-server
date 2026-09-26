package discount

type CreateDiscountDto struct {
	Code        string   `json:"code" binding:"omitempty,alphanum,min=4,max=32" example:"SUMMER30"`
	Type        string   `json:"type" binding:"required,oneof=percent value" enums:"percent,value"`
	Value       float64  `json:"value" binding:"required,gt=0" example:"30"`
	MaxPrice    *float64 `json:"max_price" binding:"omitempty,gt=0" example:"50000"`
	UseCount    int      `json:"use_count" binding:"required,min=1" example:"100"`
	Status      *bool    `json:"status"`
	OwnerMobile *string  `json:"owner_mobile" binding:"omitempty,numeric,len=11" example:"09121112233"`
}

type UpdateDiscountDto struct {
	Type        *string  `json:"type" binding:"omitempty,oneof=percent value" enums:"percent,value"`
	Value       *float64 `json:"value" binding:"omitempty,gt=0"`
	MaxPrice    *float64 `json:"max_price" binding:"omitempty,gte=0"`
	UseCount    *int     `json:"use_count" binding:"omitempty,min=1"`
	Status      *bool    `json:"status"`
	OwnerMobile *string  `json:"owner_mobile" binding:"omitempty"`
}

type ListQuery struct {
	Page    int    `form:"page,default=1" binding:"min=1"`
	Limit   int    `form:"limit,default=20" binding:"min=1,max=100"`
	Code    string `form:"code"`
	Type    string `form:"type" binding:"omitempty,oneof=percent value"`
	Status  *bool  `form:"status"`
	OwnerID uint   `form:"owner_id"`
}

func (q ListQuery) Offset() int {
	return (q.Page - 1) * q.Limit
}

type CheckDto struct {
	Code   string  `json:"code" binding:"required,max=32" example:"SUMMER30"`
	Amount float64 `json:"amount" binding:"required,gt=0" example:"200000"`
}

type CheckResponse struct {
	Code           string  `json:"code"`
	Type           string  `json:"type" enums:"percent,value"`
	Amount         float64 `json:"amount"`
	DiscountAmount float64 `json:"discount_amount"`
	PayableAmount  float64 `json:"payable_amount"`
}
