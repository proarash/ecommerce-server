package cart

type AddItemDto struct {
	ProductID uint `json:"product_id" binding:"required,min=1"`
	Quantity  int  `json:"quantity" binding:"omitempty,min=1,max=1000"`
}

type UpdateItemDto struct {
	Quantity int `json:"quantity" binding:"required,min=1,max=1000"`
}

type CartResponse struct {
	Cart
	TotalAmount float64 `json:"total_amount"`
}
