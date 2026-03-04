package domain

import "errors"

var (
	ErrNotFound     = errors.New("poll not found")
	ErrExpired      = errors.New("poll expired")
	ErrAlreadyVoted = errors.New("already voted")
	ErrInvalidOption = errors.New("invalid option")
	ErrRateLimit    = errors.New("rate limit exceeded")
)
