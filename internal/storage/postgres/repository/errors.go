package repository

import "errors"

var (
	ErrUserTeamMismatch = errors.New("user belongs to another team")
	ErrNotFound         = errors.New("record not found")
)
