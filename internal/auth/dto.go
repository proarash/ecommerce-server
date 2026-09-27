package auth

type LoginDto struct {
	Mobile   string `json:"mobile" binding:"required,numeric,len=11" example:"09120000000"`
	Password string `json:"password" binding:"required,min=6,max=72" example:"secret123"`
}

type RegisterDto struct {
	Name     *string `json:"name" binding:"omitempty,min=2,max=100"`
	Mobile   string  `json:"mobile" binding:"required,numeric,len=11" example:"09120000000"`
	Password string  `json:"password" binding:"required,min=6,max=72" example:"secret123"`
}

type TokenResponse struct {
	AccessToken string `json:"-"`
	UserID      uint   `json:"user_id"`
	Role        string `json:"role"`
	UserType    string `json:"user_type" enums:"staff,customer"`
}
