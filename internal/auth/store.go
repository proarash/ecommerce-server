package auth

import (
	"context"
	"errors"
	"time"

	"github.com/proarash/ecommerce-server/internal/staff"
	"github.com/proarash/ecommerce-server/internal/user"
	"github.com/proarash/ecommerce-server/pkg/token"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

const tokenTTL = 24 * time.Hour

var (
	ErrInvalidCredentials = errors.New("invalid mobile or password")
	ErrInactive           = errors.New("account is disabled")
	ErrMobileTaken        = errors.New("mobile already registered")
)

type AuthRepo interface {
	Login(ctx context.Context, dto LoginDto) (TokenResponse, error)
	Register(ctx context.Context, dto RegisterDto) (TokenResponse, error)
}

type authRepo struct {
	staff  staff.Store
	users  user.Store
	secret string
}

func NewAuthRepo(staffStore staff.Store, userStore user.Store, secret string) AuthRepo {
	return &authRepo{staff: staffStore, users: userStore, secret: secret}
}

func (r *authRepo) issue(id uint, mobile, role, userType string) (TokenResponse, error) {
	t, err := token.GenerateJwt(&token.AuthPayload{UserID: id, Mobile: mobile, Role: role, UserType: userType}, r.secret, tokenTTL)
	if err != nil {
		return TokenResponse{}, err
	}
	return TokenResponse{AccessToken: t, UserID: id, Role: role, UserType: userType}, nil
}

func (r *authRepo) loginStaff(ctx context.Context, dto LoginDto) (TokenResponse, error) {
	s, err := r.staff.FindByMobile(ctx, dto.Mobile)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return TokenResponse{}, ErrInvalidCredentials
		}
		return TokenResponse{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(s.Password), []byte(dto.Password)) != nil {
		return TokenResponse{}, ErrInvalidCredentials
	}
	if !s.Status {
		return TokenResponse{}, ErrInactive
	}
	return r.issue(s.ID, s.Mobile, s.Role, token.UserTypeStaff)
}

func (r *authRepo) loginCustomer(ctx context.Context, dto LoginDto) (TokenResponse, error) {
	u, err := r.users.FindByMobile(ctx, dto.Mobile)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return TokenResponse{}, ErrInvalidCredentials
		}
		return TokenResponse{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(dto.Password)) != nil {
		return TokenResponse{}, ErrInvalidCredentials
	}
	if !u.Status {
		return TokenResponse{}, ErrInactive
	}
	return r.issue(u.ID, u.Mobile, token.RoleUser, token.UserTypeCustomer)
}

func (r *authRepo) Login(ctx context.Context, dto LoginDto) (TokenResponse, error) {
	switch dto.UserType {
	case token.UserTypeStaff:
		return r.loginStaff(ctx, dto)
	case token.UserTypeCustomer:
		return r.loginCustomer(ctx, dto)
	}
	res, err := r.loginStaff(ctx, dto)
	if errors.Is(err, ErrInvalidCredentials) {
		return r.loginCustomer(ctx, dto)
	}
	return res, err
}

func (r *authRepo) Register(ctx context.Context, dto RegisterDto) (TokenResponse, error) {
	if _, err := r.users.FindByMobile(ctx, dto.Mobile); err == nil {
		return TokenResponse{}, ErrMobileTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return TokenResponse{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
	if err != nil {
		return TokenResponse{}, err
	}
	u := user.User{Name: dto.Name, Mobile: dto.Mobile, Password: string(hash), Status: true}
	if err := r.users.Create(ctx, &u); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return TokenResponse{}, ErrMobileTaken
		}
		return TokenResponse{}, err
	}
	return r.issue(u.ID, u.Mobile, token.RoleUser, token.UserTypeCustomer)
}
