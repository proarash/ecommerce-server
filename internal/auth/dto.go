package auth

type SignInDto struct {
	Mobile   string  `json:"mobile" binding:"required,numeric,min=10,max=15"`
	Otp      *string `json:"otp,omitempty" binding:"omitempty,numeric,len=5"`
	Password string  `json:"password" binding:"required_without=Otp,omitempty,min=8,max=72"`
}
