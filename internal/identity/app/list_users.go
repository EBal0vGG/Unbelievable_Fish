package app

import "context"

type ListUsers struct {
	users UserRepository
}

func NewListUsers(users UserRepository) (*ListUsers, error) {
	if users == nil {
		return nil, ErrNilUserRepository
	}
	return &ListUsers{users: users}, nil
}

func (uc *ListUsers) Execute(ctx context.Context) ([]UserDTO, error) {
	users, err := uc.users.List(ctx)
	if err != nil {
		return nil, err
	}
	result := make([]UserDTO, 0, len(users))
	for _, user := range users {
		result = append(result, userDTOFromDomain(user))
	}
	return result, nil
}
