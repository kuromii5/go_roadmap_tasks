package postgres

const (
	insertPollQuery = `
		INSERT INTO polls (id, question, expires_at)
		VALUES ($1, $2, $3)
	`

	insertOptionQuery = `
		INSERT INTO poll_options (id, poll_id, text)
		VALUES ($1, $2, $3)
	`

	selectPollQuery = `
		SELECT id, question, expires_at, created_at
		FROM polls
		WHERE id = $1
	`

	selectOptionsQuery = `
		SELECT id, poll_id, text, votes
		FROM poll_options
		WHERE poll_id = $1
		ORDER BY id
	`

	// Вставляем запись о голосе; PRIMARY KEY (poll_id, ip) отклонит дубли.
	insertVoteQuery = `
		INSERT INTO poll_votes (poll_id, ip) VALUES ($1, $2)
	`

	incrementVotesQuery = `
		UPDATE poll_options
		SET votes = votes + 1
		WHERE id = $1 AND poll_id = $2
	`

	deletePollQuery = `DELETE FROM polls WHERE id = $1`
)
