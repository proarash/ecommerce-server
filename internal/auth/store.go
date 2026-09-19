package auth

import (
	"context"
	"encoding/json"
	"errors"
	"log"

	"github.com/proarash/ecommerce-server/internal/user"
	"github.com/proarash/ecommerce-server/pkg/token"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

type AuthRepo interface {
	SignIn(dto SignInDto, ctx context.Context) string
	SignOut() bool
}

type authRepo struct {
	db *gorm.DB
}

func NewAuthRepo(db *gorm.DB) AuthRepo {
	return &authRepo{db: db}
}

// SignIn implements [AuthRepo].
func (db *authRepo) SignIn(dto SignInDto, ctx context.Context) string {
	findUser, err := gorm.G[user.User](db.db).Where("mobile = ?", dto.Mobile).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			hash, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
			if err != nil {
				log.Println(err)
			}

			var newUser user.User
			dtoBytes, _ := json.Marshal(dto)
			json.Unmarshal(dtoBytes, &newUser)

			newUser.Password = string(hash)
			newUser.Mobile = dto.Mobile

			gorm.G[user.User](db.db).Create(ctx, &newUser)
		}
	}
	err = bcrypt.CompareHashAndPassword([]byte(findUser.Password), []byte(dto.Password))
	if err != nil {
		return "Wrong password"
	}

	token, err := token.GenerateJwt(&token.AuthPayload{ID: int(findUser.ID), Mobile: findUser.Mobile})
	if err != nil {
		return err.Error()
	}

	return token
}

// SignOut implements [AuthRepo].
func (db *authRepo) SignOut() bool {
	panic("unimplemented")
}
