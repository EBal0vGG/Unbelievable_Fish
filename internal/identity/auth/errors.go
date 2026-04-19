package auth

import "errors"

var (
	ErrMissingAuthorizationHeader = errors.New("missing Authorization header")
	ErrInvalidAuthorizationHeader = errors.New("invalid Authorization header")
	ErrInvalidToken               = errors.New("invalid token")
	ErrExpiredToken               = errors.New("token expired")
	ErrIdentityNotFound           = errors.New("auth identity not found in context")
	ErrForbidden                  = errors.New("forbidden")
)
