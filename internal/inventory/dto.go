package inventory

type InboundDto struct {
	ProductID         uint    `json:"product_id" binding:"required,min=1"`
	Quantity          int     `json:"quantity" binding:"required,min=1"`
	SupplierNote      string  `json:"supplier_note" binding:"max=500"`
	WarehouseLocation *string `json:"warehouse_location" binding:"omitempty,max=100"`
}

type OutboundDto struct {
	ProductID uint   `json:"product_id" binding:"required,min=1"`
	Quantity  int    `json:"quantity" binding:"required,min=1"`
	Reason    string `json:"reason" binding:"required,max=500"`
}
