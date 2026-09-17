package auth

type SignInDto struct {
	Mobile   string `json:"mobile" binding:"required,numeric,min=11,max=11"`
	Password string `json:"password" binding:"required,min=6,max=72"`
}
