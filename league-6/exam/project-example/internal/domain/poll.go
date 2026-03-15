package domain

import "time"

type Poll struct {
	ID        string    `db:"id"`
	Question  string    `db:"question"`
	ExpiresAt time.Time `db:"expires_at"`
	CreatedAt time.Time `db:"created_at"`
}

type Option struct {
	ID     string `db:"id"`
	PollID string `db:"poll_id"`
	Text   string `db:"text"`
	Votes  int    `db:"votes"`
}

type PollResult struct {
	Poll    Poll
	Options []Option
}
