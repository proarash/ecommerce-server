package auth

import (
	"context"
	"crypto/rand"
	"crypto/sha256"
	"encoding/hex"
	"errors"
	"time"

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
	Login(ctx context.Context, dto LoginDto, device Device) (Issued, error)
	Register(ctx context.Context, dto RegisterDto, device Device) (Issued, error)
	Renew(ctx context.Context, expired *token.AuthPayload, ip string) (*token.AuthPayload, string, error)
	Revoke(ctx context.Context, p *token.AuthPayload) error
	Impersonate(ctx context.Context, adminID, userID uint) (Issued, error)
}

type Issued struct {
	AccessToken  string
	RefreshToken string
	Response     TokenResponse
}

type authRepo struct {
	db     *gorm.DB
	users  user.Store
	secret string
}

func NewAuthRepo(db *gorm.DB, userStore user.Store, secret string) AuthRepo {
	return &authRepo{db: db, users: userStore, secret: secret}
}

func newSessionID() (string, error) {
	b := make([]byte, 32)
	if _, err := rand.Read(b); err != nil {
		return "", err
	}
	return hex.EncodeToString(b), nil
}

func hashToken(t string) string {
	sum := sha256.Sum256([]byte(t))
	return hex.EncodeToString(sum[:])
}

func (r *authRepo) sign(u user.User, kind, session string, ttl time.Duration) (*token.AuthPayload, string, error) {
	p := &token.AuthPayload{UserID: u.ID, Mobile: u.Mobile, Roles: u.Roles, Kind: kind}
	p.ID = session
	t, err := token.GenerateJwt(p, r.secret, ttl)
	return p, t, err
}

func (r *authRepo) issue(ctx context.Context, u user.User, device Device) (Issued, error) {
	session, err := newSessionID()
	if err != nil {
		return Issued{}, err
	}
	_, refresh, err := r.sign(u, token.KindRefresh, session, token.RefreshTTL)
	if err != nil {
		return Issued{}, err
	}
	rt := RefreshToken{
		SessionID:  session,
		TokenHash:  hashToken(refresh),
		UserID:     u.ID,
		DeviceType: device.Type,
		UserAgent:  device.UserAgent,
		LastIP:     device.IP,
		ExpiresAt:  time.Now().Add(token.RefreshTTL),
	}
	err = r.db.WithContext(ctx).Transaction(func(tx *gorm.DB) error {
		if err := tx.Unscoped().Where("user_id = ? AND expires_at < ?", u.ID, time.Now()).Delete(&RefreshToken{}).Error; err != nil {
			return err
		}
		return tx.Create(&rt).Error
	})
	if err != nil {
		return Issued{}, err
	}
	res := TokenResponse{UserID: u.ID, Roles: u.Roles, DeviceType: device.Type}
	if device.Type == DeviceApp {
		res.RefreshToken = refresh
		res.ExpiresAt = &rt.ExpiresAt
		return Issued{RefreshToken: refresh, Response: res}, nil
	}
	_, access, err := r.sign(u, token.KindAccess, session, token.AccessTTL)
	if err != nil {
		return Issued{}, err
	}
	return Issued{AccessToken: access, Response: res}, nil
}

func (r *authRepo) Renew(ctx context.Context, expired *token.AuthPayload, ip string) (*token.AuthPayload, string, error) {
	if expired.Kind != token.KindAccess || expired.ID == "" {
		return nil, "", ErrSessionExpired
	}
	var rt RefreshToken
	err := r.db.WithContext(ctx).Where("session_id = ? AND user_id = ?", expired.ID, expired.UserID).First(&rt).Error
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
	u, err := r.users.FindByID(ctx, expired.UserID)
	if err != nil {
		return nil, "", ErrSessionExpired
	}
	if !u.Status {
		r.db.WithContext(ctx).Unscoped().Delete(&rt)
		return nil, "", ErrInactive
	}
	if ip != "" && ip != rt.LastIP {
		r.db.WithContext(ctx).Model(&rt).Update("last_ip", ip)
	}
	return r.sign(u, token.KindAccess, rt.SessionID, token.AccessTTL)
}

func (r *authRepo) Revoke(ctx context.Context, p *token.AuthPayload) error {
	return r.db.WithContext(ctx).Unscoped().Where("session_id = ? AND user_id = ?", p.ID, p.UserID).Delete(&RefreshToken{}).Error
}

func (r *authRepo) Login(ctx context.Context, dto LoginDto, device Device) (Issued, error) {
	u, err := r.users.FindByMobile(ctx, dto.Mobile)
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return Issued{}, ErrInvalidCredentials
		}
		return Issued{}, err
	}
	if bcrypt.CompareHashAndPassword([]byte(u.Password), []byte(dto.Password)) != nil {
		return Issued{}, ErrInvalidCredentials
	}
	if !u.Status {
		return Issued{}, ErrInactive
	}
	return r.issue(ctx, u, device)
}

func (r *authRepo) Register(ctx context.Context, dto RegisterDto, device Device) (Issued, error) {
	if _, err := r.users.FindByMobile(ctx, dto.Mobile); err == nil {
		return Issued{}, ErrMobileTaken
	} else if !errors.Is(err, gorm.ErrRecordNotFound) {
		return Issued{}, err
	}
	hash, err := bcrypt.GenerateFromPassword([]byte(dto.Password), bcrypt.DefaultCost)
	if err != nil {
		return Issued{}, err
	}
	u := user.User{Name: dto.Name, Mobile: dto.Mobile, Password: string(hash), Roles: user.NewRoles(), Status: true}
	if err := r.users.Create(ctx, &u); err != nil {
		if errors.Is(err, gorm.ErrDuplicatedKey) {
			return Issued{}, ErrMobileTaken
		}
		return Issued{}, err
	}
	return r.issue(ctx, u, device)
}

func (r *authRepo) Impersonate(ctx context.Context, adminID, userID uint) (Issued, error) {
	u, err := r.users.FindByID(ctx, userID)
	if err != nil {
		return Issued{}, err
	}
	p := &token.AuthPayload{UserID: u.ID, Mobile: u.Mobile, Roles: u.Roles, Kind: token.KindAccess, ImpersonatorID: &adminID}
	access, err := token.GenerateJwt(p, r.secret, token.AccessTTL)
	if err != nil {
		return Issued{}, err
	}
	return Issued{AccessToken: access, Response: TokenResponse{UserID: u.ID, Roles: u.Roles, DeviceType: DeviceBrowser}}, nil
}
