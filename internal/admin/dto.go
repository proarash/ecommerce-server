package admin

type CreateUserDto struct {
	Name            string   `json:"name" binding:"required,min=2,max=100"`
	Mobile          string   `json:"mobile" binding:"required,numeric,len=11" example:"09121112233"`
	Password        string   `json:"password" binding:"required,min=6,max=72"`
	Roles           []string `json:"roles" binding:"omitempty,dive,oneof=user storekeeper accountant marketer support" enums:"user,storekeeper,accountant,marketer,support"`
	AvatarMediaID   *uint    `json:"avatar_media_id" binding:"omitempty,min=1"`
	DefaultAvatarID *int     `json:"default_avatar_id" binding:"omitempty,min=1,max=5" enums:"1,2,3,4,5" example:"1"`
}

type UpdateStatusDto struct {
	Status *bool `json:"status" binding:"required"`
}

type UpdateRolesDto struct {
	Roles []string `json:"roles" binding:"required,dive,oneof=user storekeeper accountant marketer support" enums:"user,storekeeper,accountant,marketer,support"`
}

type StatsResponse struct {
	Users      int64   `json:"users"`
	Staff      int64   `json:"staff"`
	Products   int64   `json:"products"`
	Orders     int64   `json:"orders"`
	PaidOrders int64   `json:"paid_orders"`
	Revenue    float64 `json:"revenue"`
	OpenChats  int64   `json:"open_chats"`
	BlogPosts  int64   `json:"blog_posts"`
}
