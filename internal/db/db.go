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

func (db *Database) InsertInfraction(ctx context.Context, userID, modID, punishment, reason, appealDue, imageURL, addedRole, removedRole string) error {
	_, err := db.Pool.Exec(ctx,
		"INSERT INTO infractions (user_id, mod_id, punishment, reason, appeal_due, image_url, added_role, removed_role) VALUES ($1, $2, $3, $4, $5, $6, $7, $8)",
		userID, modID, punishment, reason, appealDue, imageURL, addedRole, removedRole,
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

func (db *Database) IsCodenameTaken(ctx context.Context, codename string) (bool, error) {
	var count int
	err := db.Pool.QueryRow(ctx, "SELECT COUNT(*) FROM codenames WHERE codename = $1 AND status = 'approved'", codename).Scan(&count)
	return count > 0, err
}

func (db *Database) InsertCodenameRequest(ctx context.Context, discordID, robloxUsername, codename string) (string, error) {
	var id string
	err := db.Pool.QueryRow(ctx,
		"INSERT INTO codenames (discord_id, roblox_username, codename, status) VALUES ($1, $2, $3, 'pending') RETURNING id",
		discordID, robloxUsername, codename,
	).Scan(&id)
	return id, err
}

func (db *Database) UpdateCodenameStatus(ctx context.Context, requestID, status string) (string, string, error) {
	var discordID, codename string
	err := db.Pool.QueryRow(ctx, "UPDATE codenames SET status = $1 WHERE id = $2 RETURNING discord_id, codename", status, requestID).Scan(&discordID, &codename)
	return discordID, codename, err
}

func (db *Database) SetAFK(ctx context.Context, userID, reason string) error {
	_, err := db.Pool.Exec(ctx,
		"INSERT INTO afk_status (user_id, reason, since) VALUES ($1, $2, now()) ON CONFLICT (user_id) DO UPDATE SET reason = $2, since = now()",
		userID, reason,
	)
	return err
}

func (db *Database) GetAFK(ctx context.Context, userID string) (string, time.Time, error) {
	var reason string
	var since time.Time
	err := db.Pool.QueryRow(ctx, "SELECT reason, since FROM afk_status WHERE user_id = $1", userID).Scan(&reason, &since)
	return reason, since, err
}

func (db *Database) RemoveAFK(ctx context.Context, userID string) error {
	_, err := db.Pool.Exec(ctx, "DELETE FROM afk_status WHERE user_id = $1", userID)
	return err
}

type Infraction struct {
	ID          string
	UserID      string
	ModID       string
	Punishment  string
	Reason      string
	AppealDue   *string
	ImageURL    *string
	AddedRole   *string
	RemovedRole *string
	CreatedAt   time.Time
}

func (db *Database) GetInfractions(ctx context.Context, userID string) ([]Infraction, error) {
	rows, err := db.Pool.Query(ctx, "SELECT id, user_id, mod_id, punishment, reason, appeal_due, image_url, added_role, removed_role, created_at FROM infractions WHERE user_id = $1 ORDER BY created_at DESC", userID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Infraction
	for rows.Next() {
		var inf Infraction
		err := rows.Scan(&inf.ID, &inf.UserID, &inf.ModID, &inf.Punishment, &inf.Reason, &inf.AppealDue, &inf.ImageURL, &inf.AddedRole, &inf.RemovedRole, &inf.CreatedAt)
		if err != nil {
			return nil, err
		}
		list = append(list, inf)
	}
	return list, rows.Err()
}

func (db *Database) InsertLoaRoaRequest(ctx context.Context, userID, requestType, fromWhen, tillWhen, reason string) (string, error) {
	var id string
	err := db.Pool.QueryRow(ctx,
		"INSERT INTO loa_roa (user_id, type, from_when, till_when, reason, status) VALUES ($1, $2, $3, $4, $5, 'pending') RETURNING id",
		userID, requestType, fromWhen, tillWhen, reason,
	).Scan(&id)
	return id, err
}

func (db *Database) GetLoaRoaRequest(ctx context.Context, id string) (string, string, string, error) {
	var userID, reqType, tillWhen string
	err := db.Pool.QueryRow(ctx, "SELECT user_id, type, till_when FROM loa_roa WHERE id = $1", id).Scan(&userID, &reqType, &tillWhen)
	return userID, reqType, tillWhen, err
}

func (db *Database) UpdateLoaRoaStatus(ctx context.Context, requestID, status string, expiresAt *time.Time) (string, string, error) {
	var userID, requestType string
	err := db.Pool.QueryRow(ctx,
		"UPDATE loa_roa SET status = $1, expires_at = $2 WHERE id = $3 RETURNING user_id, type",
		status, expiresAt, requestID,
	).Scan(&userID, &requestType)
	return userID, requestType, err
}

type ExpiredLeave struct {
	ID     string
	UserID string
	Type   string
}

func (db *Database) GetExpiredLeaves(ctx context.Context) ([]ExpiredLeave, error) {
	rows, err := db.Pool.Query(ctx, "SELECT id, user_id, type FROM loa_roa WHERE status = 'approved' AND expires_at <= now() AND notified = FALSE")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []ExpiredLeave
	for rows.Next() {
		var el ExpiredLeave
		err := rows.Scan(&el.ID, &el.UserID, &el.Type)
		if err != nil {
			return nil, err
		}
		list = append(list, el)
	}
	return list, rows.Err()
}

func (db *Database) MarkLeaveNotified(ctx context.Context, id string) error {
	_, err := db.Pool.Exec(ctx, "UPDATE loa_roa SET notified = TRUE WHERE id = $1", id)
	return err
}

func (db *Database) UpsertUserRoles(ctx context.Context, userID string, roleIDs []string) error {
	_, err := db.Pool.Exec(ctx,
		"INSERT INTO user_roles (user_id, role_ids, updated_at) VALUES ($1, $2, now()) ON CONFLICT (user_id) DO UPDATE SET role_ids = $2, updated_at = now()",
		userID, roleIDs,
	)
	return err
}

func (db *Database) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	var roleIDs []string
	err := db.Pool.QueryRow(ctx, "SELECT role_ids FROM user_roles WHERE user_id = $1", userID).Scan(&roleIDs)
	return roleIDs, err
}
