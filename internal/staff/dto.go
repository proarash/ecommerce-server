package staff

type UpdateProfileDto struct {
	Name            *string `json:"name" binding:"omitempty,min=2,max=100"`
	AvatarMediaID   *uint   `json:"avatar_media_id" binding:"omitempty,min=1"`
	DefaultAvatarID *int    `json:"default_avatar_id" binding:"omitempty,min=1,max=5" enums:"1,2,3,4,5" example:"1"`
}
