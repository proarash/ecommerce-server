package auth

import (
	"context"
	"crypto/sha256"
	"encoding/hex"
	"encoding/json/v2"
	"errors"
	"log"

	"github.com/proarash/ecommerce-server/internal/user"
	"gorm.io/gorm"
)

type AuthRepo interface {
	SignIn(dto SignInDto, ctx context.Context) bool
	SignOut() bool
}

type authRepo struct {
	db *gorm.DB
}

func NewAuthRepo(db *gorm.DB) AuthRepo {
	return &authRepo{db: db}
}

// SignIn implements [AuthRepo].
func (db *authRepo) SignIn(dto SignInDto, ctx context.Context) bool {
	findUser, err := gorm.G[user.User](db.db).Where("mobile = ?", dto.Mobile).First(ctx)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			log.Println("Record not found")

			hash := sha256.Sum256([]byte(dto.Password))
			hashPwd := hex.EncodeToString(hash[:])

			var newUser user.User
			dtoBytes, _ := json.Marshal(dto)
			json.Unmarshal(dtoBytes, &newUser)

			newUser.Password = hashPwd

			err := gorm.G[user.User](db.db).Create(ctx, &newUser)
			if err != nil {
				log.Println(err)
			}
		}
		log.Println("database error")
	}
	log.Println(findUser.Mobile)

	return true
}

// SignOut implements [AuthRepo].
func (db *authRepo) SignOut() bool {
	panic("unimplemented")
}
