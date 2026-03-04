package poll

import (
	"context"

	"github.com/kuromii5/snapbin/internal/domain"
)

type PollRepo interface {
	SavePoll(ctx context.Context, poll *domain.Poll, options []*domain.Option) error
	GetByID(ctx context.Context, id string) (*domain.PollResult, error)
	Vote(ctx context.Context, pollID, optionID, ip string) error
	Delete(ctx context.Context, id string) error
}

type Cache interface {
	Get(ctx context.Context, id string) (*domain.PollResult, error)
	Set(ctx context.Context, result *domain.PollResult) error
	Delete(ctx context.Context, id string) error
}

type Service struct {
	repo  PollRepo
	cache Cache
}

func NewService(repo PollRepo, cache Cache) *Service {
	return &Service{repo: repo, cache: cache}
}
