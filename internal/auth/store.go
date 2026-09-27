package auth

import (
	"context"
	"crypto/rand"
	"encoding/hex"
	"errors"
	"time"

	"github.com/proarash/ecommerce-server/internal/staff"
	"github.com/proarash/ecommerce-server/internal/user"
	"github.com/proarash/ecommerce-server/pkg/token"
	"golang.org/x/crypto/bcrypt"
	"gorm.io/gorm"
)

var (
	ErrInvalidCredentials = errors.New("invalid mobile or password")
	ErrInactive           = errors.New("account is disabled")
	ErrMobileTaken        = errors.New("mobile already registered")
	ErrSessionExpired     = errors.New("session expired")
)

type AuthRepo interface {
	LoginStaff(ctx context.Context, dto LoginDto) (TokenResponse, error)
	LoginCustomer(ctx context.Context, dto LoginDto) (TokenResponse, error)
	Register(ctx context.Context, dto RegisterDto) (TokenResponse, error)
	Refresh(ctx context.Context, expired *token.AuthPayload) (*token.AuthPayload, string, error)
	Revoke(ctx context.Context, p *token.AuthPayload) error
	Impersonate(ctx context.Context, adminID, userID uint) (TokenResponse, error)
}

type authRepo struct {
	db     *gorm.DB
	staff  staff.Store
	users  user.Store
	secret string
}

func NewAuthRepo(db *gorm.DB, staffStore staff.Store, userStore user.Store, secret string) AuthRepo {
	return &authRepo{db: db, staff: staffStore, users: userStore, secret: secret}
}

func ownerColumn(userType string) string {
	if userType == token.UserTypeStaff {
		return "staff_id"
	}
	return "user_id"
}

func newSessionToken() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func (r *authRepo) accessToken(id uint, mobile, role, userType, session string) (*token.AuthPayload, string, error) {
	p := &token.AuthPayload{UserID: id, Mobile: mobile, Role: role, UserType: userType}
	p.ID = session
	t, err := token.GenerateJwt(p, r.secret, token.AccessTTL)
	return p, t, err
}

func (r *authRepo) issue(ctx context.Context, id uint, mobile, role, userType string) (TokenResponse, error) {
	session, err := newSessionToken()
	if err != nil {
		return TokenResponse{}, err
	}
	rt := RefreshToken{Token: session, ExpiresAt: time.Now().Add(token.RefreshTTL)}
	if userType == token.UserTypeStaff {
		rt.StaffID = &id
	} else {
		rt.UserID = &id
	}
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where(ownerColumn(userType)+" = ?", id).Delete(&RefreshToken{}).Error; err != nil {
			return err
		}
		return tx.Create(&rt).Error
	})
	if err != nil {
		return TokenResponse{}, err
	}
	_, access, err := r.accessToken(id, mobile, role, userType, session)
	if err != nil {
		return TokenResponse{}, err
	}
	return TokenResponse{AccessToken: access, UserID: id, Role: role, UserType: userType}, nil
}

func (r *authRepo) Refresh(ctx context.Context, expired *token.AuthPayload) (*token.AuthPayload, string, error) {
	var rt RefreshToken
	err := r.db.WithContext(ctx).Where(ownerColumn(expired.UserType)+" = ? AND token = ?", expired.UserID, expired.ID).First(&rt).Error
	if errors.Is(err, gorm.ErrRecordNotFound) {
		return nil, "", ErrSessionExpired
	}
	if err != nil {
		return nil, "", err
	}
	if time.Now().After(rt.ExpiresAt) {
		r.db.WithContext(ctx).Unscoped().Delete(&rt)
		return nil, "", ErrSessionExpired
	}
	var mobile, role string
	var active bool
	if expired.UserType == token.UserTypeStaff {
		s, err := r.staff.FindByID(ctx, expired.UserID)
		if err != nil {
			return nil, "", ErrSessionExpired
		}
		mobile, role, active = s.Mobile, s.Role, s.Status
	} else {
		u, err := r.users.FindByID(ctx, expired.UserID)
		if err != nil {
			return nil, "", ErrSessionExpired
		}
		mobile, role, active = u.Mobile, token.RoleUser, u.Status
	}
	if !active {
		r.db.WithContext(ctx).Unscoped().Delete(&rt)
		return nil, "", ErrInactive
	}
	return r.accessToken(expired.UserID, mobile, role, expired.UserType, rt.Token)
}

func (r *authRepo) Revoke(ctx context.Context, p *token.AuthPayload) error {
	return r.db.WithContext(ctx).Unscoped().Where(ownerColumn(p.UserType)+" = ? AND token = ?", p.UserID, p.ID).Delete(&RefreshToken{}).Error
}

func (r *authRepo) LoginStaff(ctx context.Context, dto LoginDto) (TokenResponse, error) {
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
	return r.issue(ctx, s.ID, s.Mobile, s.Role, token.UserTypeStaff)
}

func (r *authRepo) LoginCustomer(ctx context.Context, dto LoginDto) (TokenResponse, error) {
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
	return r.issue(ctx, u.ID, u.Mobile, token.RoleUser, token.UserTypeCustomer)
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
	return r.issue(ctx, u.ID, u.Mobile, token.RoleUser, token.UserTypeCustomer)
}

func (r *authRepo) Impersonate(ctx context.Context, adminID, userID uint) (TokenResponse, error) {
	u, err := r.users.FindByID(ctx, userID)
	if err != nil {
		return TokenResponse{}, err
	}
	p := &token.AuthPayload{UserID: u.ID, Mobile: u.Mobile, Role: token.RoleUser, UserType: token.UserTypeCustomer, ImpersonatorID: &adminID}
	access, err := token.GenerateJwt(p, r.secret, token.AccessTTL)
	if err != nil {
		return TokenResponse{}, err
	}
	return TokenResponse{AccessToken: access, UserID: u.ID, Role: token.RoleUser, UserType: token.UserTypeCustomer}, nil
}
