package notification

type CreateNotificationDto struct {
	UserID  *uint  `json:"user_id" binding:"omitempty,min=1"`
	Title   string `json:"title" binding:"required,max=200"`
	Message string `json:"message" binding:"required"`
	Type    string `json:"type" binding:"required,oneof=order payment system promo" enums:"order,payment,system,promo"`
	Channel string `json:"channel" binding:"required,oneof=in_app telegram both" enums:"in_app,telegram,both"`
}

type UpdateNotificationDto struct {
	Title   *string `json:"title" binding:"omitempty,max=200"`
	Message *string `json:"message"`
	Type    *string `json:"type" binding:"omitempty,oneof=order payment system promo" enums:"order,payment,system,promo"`
	Channel *string `json:"channel" binding:"omitempty,oneof=in_app telegram both" enums:"in_app,telegram,both"`
	IsRead  *bool   `json:"is_read"`
}

type ListQuery struct {
	Page   int   `form:"page,default=1" binding:"min=1"`
	Limit  int   `form:"limit,default=20" binding:"min=1,max=100"`
	UserID *uint `form:"user_id"`
}
