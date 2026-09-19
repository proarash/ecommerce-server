package token

import (
	"os"

	"github.com/golang-jwt/jwt/v5"
)

type AuthPayload struct {
	ID     int
	Mobile string
	jwt.RegisteredClaims
}

func GenerateJwt(payload *AuthPayload) (string, error) {
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	tokenStr, err := token.SignedString([]byte(os.Getenv("JWT_SECRET")))
	if err != nil {
		return "", err
	}
	return tokenStr, nil
}
