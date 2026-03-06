package poll

import (
	"context"
	"errors"
	"fmt"
	"time"

	"github.com/google/uuid"

	"github.com/kuromii5/poller/internal/domain"
)

const (
	defaultTTLMinutes = 60
	maxTTLMinutes     = 7 * 24 * 60
	minOptions        = 2
	maxOptions        = 10
)

type CreateInput struct {
	Question   string
	Options    []string
	TTLMinutes int
}

type CreateResult struct {
	ID        string
	Link      string
	ExpiresAt time.Time
}

func (s *Service) Create(ctx context.Context, in CreateInput) (*CreateResult, error) {
	if in.Question == "" {
		return nil, errors.New("question is required")
	}
	if len(in.Options) < minOptions {
		return nil, fmt.Errorf("at least %d options required", minOptions)
	}
	if len(in.Options) > maxOptions {
		return nil, fmt.Errorf("at most %d options allowed", maxOptions)
	}
	for _, opt := range in.Options {
		if opt == "" {
			return nil, errors.New("option text cannot be empty")
		}
	}

	if in.TTLMinutes <= 0 {
		in.TTLMinutes = defaultTTLMinutes
	}
	if in.TTLMinutes > maxTTLMinutes {
		return nil, fmt.Errorf("ttl_minutes must be at most %d", maxTTLMinutes)
	}

	pollID := uuid.New().String()
	expiresAt := time.Now().Add(time.Duration(in.TTLMinutes) * time.Minute)

	poll := &domain.Poll{
		ID:        pollID,
		Question:  in.Question,
		ExpiresAt: expiresAt,
	}

	options := make([]*domain.Option, len(in.Options))
	for i, text := range in.Options {
		options[i] = &domain.Option{
			ID:     uuid.New().String(),
			PollID: pollID,
			Text:   text,
		}
	}

	if err := s.repo.SavePoll(ctx, poll, options); err != nil {
		return nil, fmt.Errorf("save poll: %w", err)
	}

	return &CreateResult{
		ID:        pollID,
		Link:      "/api/polls/" + pollID,
		ExpiresAt: expiresAt,
	}, nil
}
