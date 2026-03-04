package postgres

import (
	"context"
	"database/sql"
	"errors"
	"fmt"

	"github.com/jackc/pgx/v5/pgconn"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/jmoiron/sqlx"

	"github.com/kuromii5/snapbin/config"
	"github.com/kuromii5/snapbin/internal/domain"
)

type DB struct {
	*sqlx.DB
}

func New(cfg config.DBConfig) (*DB, error) {
	db, err := sqlx.Connect("pgx", cfg.DSN())
	if err != nil {
		return nil, fmt.Errorf("connect to postgres: %w", err)
	}
	return &DB{db}, nil
}

func (db *DB) SavePoll(ctx context.Context, poll *domain.Poll, options []*domain.Option) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	if _, err := tx.ExecContext(ctx, insertPollQuery, poll.ID, poll.Question, poll.ExpiresAt); err != nil {
		return fmt.Errorf("insert poll: %w", err)
	}

	for _, opt := range options {
		if _, err := tx.ExecContext(ctx, insertOptionQuery, opt.ID, opt.PollID, opt.Text); err != nil {
			return fmt.Errorf("insert option: %w", err)
		}
	}

	return tx.Commit()
}

func (db *DB) GetByID(ctx context.Context, id string) (*domain.PollResult, error) {
	var poll domain.Poll
	err := db.GetContext(ctx, &poll, selectPollQuery, id)
	if errors.Is(err, sql.ErrNoRows) {
		return nil, domain.ErrNotFound
	}
	if err != nil {
		return nil, fmt.Errorf("get poll: %w", err)
	}

	var options []domain.Option
	if err := db.SelectContext(ctx, &options, selectOptionsQuery, id); err != nil {
		return nil, fmt.Errorf("get options: %w", err)
	}

	return &domain.PollResult{Poll: poll, Options: options}, nil
}

func (db *DB) Vote(ctx context.Context, pollID, optionID, ip string) error {
	tx, err := db.BeginTxx(ctx, nil)
	if err != nil {
		return fmt.Errorf("begin tx: %w", err)
	}
	defer tx.Rollback()

	_, err = tx.ExecContext(ctx, insertVoteQuery, pollID, ip)
	if err != nil {
		var pgErr *pgconn.PgError
		if errors.As(err, &pgErr) && pgErr.Code == "23505" {
			return domain.ErrAlreadyVoted
		}
		return fmt.Errorf("insert vote: %w", err)
	}

	res, err := tx.ExecContext(ctx, incrementVotesQuery, optionID, pollID)
	if err != nil {
		return fmt.Errorf("increment votes: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrInvalidOption
	}

	return tx.Commit()
}

func (db *DB) Delete(ctx context.Context, id string) error {
	res, err := db.ExecContext(ctx, deletePollQuery, id)
	if err != nil {
		return fmt.Errorf("delete poll: %w", err)
	}
	if n, _ := res.RowsAffected(); n == 0 {
		return domain.ErrNotFound
	}
	return nil
}
