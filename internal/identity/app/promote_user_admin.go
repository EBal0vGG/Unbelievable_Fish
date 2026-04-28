package app

import (
	"context"
	"strings"
)

type PromoteUserToAdmin struct {
	users UserRepository
}

func NewPromoteUserToAdmin(users UserRepository) (*PromoteUserToAdmin, error) {
	if users == nil {
		return nil, ErrNilUserRepository
	}
	return &PromoteUserToAdmin{users: users}, nil
}

func (uc *PromoteUserToAdmin) Execute(ctx context.Context, cmd PromoteUserToAdminCommand) (UserDTO, error) {
	userID := strings.TrimSpace(cmd.UserID)
	if userID == "" {
		return UserDTO{}, ErrUserIDRequired
	}
	user, err := uc.users.GetByID(ctx, userID)
	if err != nil {
		return UserDTO{}, err
	}
	user.PromoteToAdmin()
	if err := uc.users.Save(ctx, user); err != nil {
		return UserDTO{}, err
	}
	return userDTOFromDomain(user), nil
}
