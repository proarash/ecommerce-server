package auth

import "gorm.io/gorm"

type AuthRepo interface {
	SignIn(dto SignInDto) bool
	SignOut() bool
}

type authRepo struct {
	db *gorm.DB
}

func NewAuthRepo(db *gorm.DB) AuthRepo {
	return &authRepo{db: db}
}

// SignIn implements [AuthRepo].
func (db *authRepo) SignIn(dto SignInDto) bool {
	return true
}

// SignOut implements [AuthRepo].
func (db *authRepo) SignOut() bool {
	panic("unimplemented")
}
