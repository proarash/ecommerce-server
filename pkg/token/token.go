package token

import (
	"errors"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	UserTypeStaff    = "staff"
	UserTypeCustomer = "customer"
	RoleUser         = "user"
	AccessTTL        = 2 * time.Hour
	RefreshTTL       = 90 * 24 * time.Hour
	StaffCookie      = "staff_access_token"
	CustomerCookie   = "user_access_token"
)

type AuthPayload struct {
	UserID         uint   `json:"user_id"`
	Mobile         string `json:"mobile"`
	Role           string `json:"role"`
	UserType       string `json:"user_type"`
	ImpersonatorID *uint  `json:"impersonator_id,omitempty"`
	jwt.RegisteredClaims
}

func GenerateJwt(payload *AuthPayload, secret string, ttl time.Duration) (string, error) {
	now := time.Now()
	payload.IssuedAt = jwt.NewNumericDate(now)
	payload.ExpiresAt = jwt.NewNumericDate(now.Add(ttl))
	token := jwt.NewWithClaims(jwt.SigningMethodHS256, payload)
	return token.SignedString([]byte(secret))
}

func parse(tokenStr, secret string, opts ...jwt.ParserOption) (*AuthPayload, error) {
	opts = append(opts, jwt.WithValidMethods([]string{jwt.SigningMethodHS256.Alg()}))
	token, err := jwt.ParseWithClaims(tokenStr, &AuthPayload{}, func(t *jwt.Token) (any, error) {
		return []byte(secret), nil
	}, opts...)
	if err != nil {
		return nil, err
	}
	claims, ok := token.Claims.(*AuthPayload)
	if !ok || !token.Valid {
		return nil, errors.New("invalid token")
	}
	return claims, nil
}

func ParseJwt(tokenStr, secret string) (*AuthPayload, error) {
	return parse(tokenStr, secret)
}

func ParseExpiredJwt(tokenStr, secret string) (*AuthPayload, error) {
	return parse(tokenStr, secret, jwt.WithoutClaimsValidation())
}
