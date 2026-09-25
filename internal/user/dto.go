package user

type UpdateProfileDto struct {
	Name           *string  `json:"name" binding:"omitempty,min=2,max=100"`
	Mobile         *string  `json:"mobile" binding:"omitempty,numeric,len=11"`
	TelegramChatID *int64   `json:"telegram_chat_id"`
	Address        *string  `json:"address" binding:"omitempty,max=500"`
	Latitude       *float64 `json:"latitude" binding:"omitempty,latitude"`
	Longitude      *float64 `json:"longitude" binding:"omitempty,longitude"`
}
