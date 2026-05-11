package app

import (
	"context"
	"errors"
	"strings"
)

type Login struct {
	users  UserRepository
	hasher PasswordHasher
	tokens TokenProvider
}

func NewLogin(users UserRepository, hasher PasswordHasher, tokens TokenProvider) (*Login, error) {
	if users == nil {
		return nil, ErrNilUserRepository
	}
	if hasher == nil {
		return nil, ErrNilPasswordHasher
	}
	if tokens == nil {
		return nil, ErrNilTokenProvider
	}
	return &Login{
		users:  users,
		hasher: hasher,
		tokens: tokens,
	}, nil
}

func (uc *Login) Execute(ctx context.Context, cmd LoginCommand) (LoginResult, error) {
	login := strings.ToLower(strings.TrimSpace(cmd.Login))
	if strings.TrimSpace(cmd.Password) == "" {
		return LoginResult{}, ErrPasswordRequired
	}

	user, err := uc.users.GetByLogin(ctx, login)
	if err != nil {
		if errors.Is(err, ErrUserNotFound) {
			return LoginResult{}, ErrInvalidCredentials
		}
		return LoginResult{}, err
	}

	ok, err := uc.hasher.Compare(user.PasswordHash(), cmd.Password)
	if err != nil {
		return LoginResult{}, err
	}
	if !ok {
		return LoginResult{}, ErrInvalidCredentials
	}
	if !user.EmailVerified() {
		return LoginResult{}, ErrEmailNotVerified
	}

	token, err := uc.tokens.Generate(user)
	if err != nil {
		return LoginResult{}, err
	}

	return LoginResult{
		Token: token,
		User:  userDTOFromDomain(user),
	}, nil
}
