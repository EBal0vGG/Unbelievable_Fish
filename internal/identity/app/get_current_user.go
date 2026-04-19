package app

import (
	"context"
	"strings"
)

type GetCurrentUser struct {
	users UserRepository
}

func NewGetCurrentUser(users UserRepository) (*GetCurrentUser, error) {
	if users == nil {
		return nil, ErrNilUserRepository
	}
	return &GetCurrentUser{
		users: users,
	}, nil
}

func (uc *GetCurrentUser) Execute(ctx context.Context, query GetCurrentUserQuery) (UserDTO, error) {
	if strings.TrimSpace(query.UserID) == "" {
		return UserDTO{}, ErrUserIDRequired
	}
	user, err := uc.users.GetByID(ctx, query.UserID)
	if err != nil {
		return UserDTO{}, err
	}
	return userDTOFromDomain(user), nil
}
