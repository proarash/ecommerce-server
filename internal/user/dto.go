package user

type UpdateProfileDto struct {
	Name            *string  `json:"name" binding:"omitempty,min=2,max=100"`
	Mobile          *string  `json:"mobile" binding:"omitempty,numeric,len=11"`
	AvatarMediaID   *uint    `json:"avatar_media_id" binding:"omitempty,min=1"`
	DefaultAvatarID *int     `json:"default_avatar_id" binding:"omitempty,min=1,max=5" enums:"1,2,3,4,5" example:"1"`
	TelegramChatID  *int64   `json:"telegram_chat_id"`
	Address         *string  `json:"address" binding:"omitempty,max=500"`
	Latitude        *float64 `json:"latitude" binding:"omitempty,latitude"`
	Longitude       *float64 `json:"longitude" binding:"omitempty,longitude"`
	Password        *string  `json:"password" binding:"omitempty,min=6,max=72" example:"secret123"`
}

type ListQuery struct {
	Page   int    `form:"page,default=1" binding:"min=1"`
	Limit  int    `form:"limit,default=20" binding:"min=1,max=100"`
	Role   string `form:"role" binding:"omitempty,oneof=user admin storekeeper accountant marketer support"`
	Staff  bool   `form:"staff"`
	Search string `form:"search"`
}
