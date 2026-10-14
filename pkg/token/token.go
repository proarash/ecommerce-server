package token

import (
	"errors"
	"slices"
	"time"

	"github.com/golang-jwt/jwt/v5"
)

const (
	AccessTTL   = 24 * time.Hour
	RefreshTTL  = 90 * 24 * time.Hour
	KindAccess  = "access"
	KindRefresh = "refresh"
)

type AuthPayload struct {
	UserID         uint     `json:"user_id"`
	Mobile         string   `json:"mobile"`
	Roles          []string `json:"roles"`
	Kind           string   `json:"kind"`
	ImpersonatorID *uint    `json:"impersonator_id,omitempty"`
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

func IsExpired(err error) bool {
	return errors.Is(err, jwt.ErrTokenExpired)
}

func (p *AuthPayload) HasRole(role string) bool {
	return slices.Contains(p.Roles, role)
}
