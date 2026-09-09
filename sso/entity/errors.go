package entity

import "errors"

var (
	ErrUserExist          = errors.New("user already exist")
	ErrInvalidCredentials = errors.New("invalid data")
	ErrUserNotFound       = errors.New("user not found")
	ErrInvalidToken       = errors.New("invalid Token")
)
