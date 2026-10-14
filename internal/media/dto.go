package media

const (
	TargetProduct  = 1
	TargetCategory = 2
	TargetBanner   = 3
)

type UploadQuery struct {
	PID  uint `form:"pid" binding:"required_with=Type,omitempty,min=1"`
	Type int  `form:"type" binding:"required_with=PID,omitempty,oneof=1 2 3"`
}

type ListQuery struct {
	Page  int    `form:"page,default=1" binding:"min=1"`
	Limit int    `form:"limit,default=20" binding:"min=1,max=100"`
	ID    uint   `form:"id"`
	Type  string `form:"mime" binding:"omitempty,oneof=image/webp image/svg+xml"`
}
