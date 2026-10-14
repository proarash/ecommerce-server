package media

const (
	TargetProduct  = 1
	TargetCategory = 2
)

type UploadQuery struct {
	PID  uint `form:"pid" binding:"required,min=1"`
	Type int  `form:"type" binding:"required,oneof=1 2"`
}
