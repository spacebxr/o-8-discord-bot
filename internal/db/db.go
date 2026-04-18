package db

import (
	"context"
	"time"

	"github.com/jackc/pgx/v5/pgxpool"
)

type Database struct {
	Pool *pgxpool.Pool
}

func Connect(ctx context.Context, dbURL string) (*Database, error) {
	pool, err := pgxpool.New(ctx, dbURL)
	if err != nil {
		return nil, err
	}
	return &Database{Pool: pool}, nil
}

func (db *Database) InsertInfraction(ctx context.Context, userID, modID string, severity int, reason, whatPunishment, tillWhen string) error {
	_, err := db.Pool.Exec(ctx,
		"INSERT INTO infractions (user_id, mod_id, severity, reason, what_punishment, till_when) VALUES ($1, $2, $3, $4, $5, $6)",
		userID, modID, severity, reason, whatPunishment, tillWhen,
	)
	return err
}

func (db *Database) CountInfractions(ctx context.Context, userID string) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM infractions WHERE user_id = $1", userID).Scan(&count)
	return count, err
}

func (db *Database) StartStopwatch(ctx context.Context, userID string) error {
	_, err := db.Pool.Exec(ctx,
		"INSERT INTO stopwatches (user_id, start_time) VALUES ($1, now()) ON CONFLICT (user_id) DO UPDATE SET start_time = now() WHERE stopwatches.start_time IS NULL",
		userID,
	)
	return err
}

func (db *Database) StopStopwatch(ctx context.Context, userID string) (int64, error) {
	var total int64
	err := db.Pool.QueryRow(ctx,
		"UPDATE stopwatches SET total_seconds = total_seconds + EXTRACT(EPOCH FROM (now() - start_time)), start_time = NULL WHERE user_id = $1 AND start_time IS NOT NULL RETURNING total_seconds",
		userID,
	).Scan(&total)
	return total, err
}

func (db *Database) GetStopwatch(ctx context.Context, userID string) (*time.Time, int64, error) {
	var startTime *time.Time
	var totalSeconds int64
	err := db.Pool.QueryRow(ctx, "SELECT start_time, total_seconds FROM stopwatches WHERE user_id = $1", userID).Scan(&startTime, &totalSeconds)
	if err != nil {
		return nil, 0, err
	}
	return startTime, totalSeconds, nil
}

func (db *Database) ResetStopwatch(ctx context.Context, userID string) error {
	_, err := db.Pool.Exec(ctx, "DELETE FROM stopwatches WHERE user_id = $1", userID)
	return err
}
