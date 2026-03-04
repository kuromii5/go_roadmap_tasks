package poll

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/kuromii5/snapbin/internal/domain"
)

func (s *Service) Get(ctx context.Context, id string) (*domain.PollResult, error) {
	result, err := s.cache.Get(ctx, id)
	if errors.Is(err, domain.ErrNotFound) {
		result, err = s.repo.GetByID(ctx, id)
	}
	if err != nil {
		return nil, err
	}

	if time.Now().After(result.Poll.ExpiresAt) {
		_ = s.repo.Delete(ctx, id)
		_ = s.cache.Delete(ctx, id)
		return nil, domain.ErrExpired
	}

	// Populate cache on miss
	_ = s.cache.Set(ctx, result)

	return result, nil
}

func (s *Service) Vote(ctx context.Context, pollID, optionID, ip string) error {
	result, err := s.Get(ctx, pollID)
	if err != nil {
		return err
	}

	validOption := false
	for _, opt := range result.Options {
		if opt.ID == optionID {
			validOption = true
			break
		}
	}
	if !validOption {
		return domain.ErrInvalidOption
	}

	if err := s.repo.Vote(ctx, pollID, optionID, ip); err != nil {
		return fmt.Errorf("vote: %w", err)
	}

	// Invalidate cache so next Get returns fresh vote counts
	_ = s.cache.Delete(ctx, pollID)

	return nil
}
