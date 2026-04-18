package db

import (
	"context"

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
