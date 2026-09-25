package admin

type CreateStaffDto struct {
	Name            string `json:"name" binding:"required,min=2,max=100"`
	Mobile          string `json:"mobile" binding:"required,numeric,len=11" example:"09121112233"`
	Password        string `json:"password" binding:"required,min=6,max=72"`
	Role            string `json:"role" binding:"required,oneof=storekeeper accountant marketer support" enums:"storekeeper,accountant,marketer,support"`
	AvatarMediaID   *uint  `json:"avatar_media_id" binding:"omitempty,min=1"`
	DefaultAvatarID *int   `json:"default_avatar_id" binding:"omitempty,min=1,max=5" enums:"1,2,3,4,5" example:"1"`
}

type UpdateStaffStatusDto struct {
	Status *bool `json:"status" binding:"required"`
}

type StaffQuery struct {
	Page  int    `form:"page,default=1" binding:"min=1"`
	Limit int    `form:"limit,default=20" binding:"min=1,max=100"`
	Role  string `form:"role" binding:"omitempty,oneof=admin storekeeper accountant marketer support"`
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
