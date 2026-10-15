package user

import (
	"context"

	"golang.org/x/crypto/bcrypt"
)

type Service struct {
	store Store
}

func NewService(store Store) *Service {
	return &Service{store: store}
}

func (s *Service) Profile(ctx context.Context, id uint) (User, error) {
	return s.store.FindByID(ctx, id)
}

func (s *Service) UpdateProfile(ctx context.Context, id uint, dto UpdateProfileDto) (User, error) {
	fields := map[string]any{}
	if dto.Name != nil {
		fields["name"] = *dto.Name
	}
	if dto.AvatarMediaID != nil {
		fields["avatar_media_id"] = *dto.AvatarMediaID
	}
	if dto.DefaultAvatarID != nil {
		fields["default_avatar_id"] = *dto.DefaultAvatarID
	}
	if dto.Mobile != nil {
		fields["mobile"] = *dto.Mobile
	}
	if dto.TelegramChatID != nil {
		fields["telegram_chat_id"] = *dto.TelegramChatID
	}
	if dto.Address != nil {
		fields["address"] = *dto.Address
	}
	if dto.Latitude != nil {
		fields["latitude"] = *dto.Latitude
	}
	if dto.Longitude != nil {
		fields["longitude"] = *dto.Longitude
	}
	if dto.Password != nil {
		hash, err := bcrypt.GenerateFromPassword([]byte(*dto.Password), bcrypt.DefaultCost)
		if err != nil {
			return User{}, err
		}
		fields["password"] = string(hash)
	}
	if len(fields) > 0 {
		if err := s.store.Update(ctx, id, fields); err != nil {
			return User{}, err
		}
	}
	return s.store.FindByID(ctx, id)
}
