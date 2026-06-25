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

func (db *Database) StopStopwatch(ctx context.Context, userID string) (int64, time.Time, time.Time, error) {
	var total int64
	var startedAt, endedAt time.Time
	err := db.Pool.QueryRow(ctx,
		`WITH prev AS (
			SELECT start_time FROM stopwatches WHERE user_id = $1 AND start_time IS NOT NULL
		), upd AS (
			UPDATE stopwatches SET total_seconds = total_seconds + EXTRACT(EPOCH FROM (now() - start_time))::bigint, start_time = NULL WHERE user_id = $1 AND start_time IS NOT NULL RETURNING total_seconds
		)
		SELECT upd.total_seconds, prev.start_time, now() FROM upd, prev`,
		userID,
	).Scan(&total, &startedAt, &endedAt)
	return total, startedAt, endedAt, err
}

func (db *Database) LogStopwatchSession(ctx context.Context, userID string, startedAt, endedAt time.Time, durationSeconds int64) error {
	_, err := db.Pool.Exec(ctx,
		"INSERT INTO stopwatch_sessions (user_id, started_at, ended_at, duration_seconds) VALUES ($1, $2, $3, $4)",
		userID, startedAt, endedAt, durationSeconds,
	)
	return err
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

func (db *Database) GetActiveUserLOA(ctx context.Context, userID string) (*time.Time, error) {
	var expiresAt *time.Time
	err := db.Pool.QueryRow(ctx, "SELECT expires_at FROM loa_roa WHERE user_id = $1 AND type = 'LOA' AND status = 'approved' AND (expires_at > now() OR expires_at IS NULL) ORDER BY expires_at DESC NULLS LAST LIMIT 1", userID).Scan(&expiresAt)
	return expiresAt, err
}

func (db *Database) UpsertUserRoles(ctx context.Context, userID string, roleIDs []string) error {
	_, err := db.Pool.Exec(ctx,
		"INSERT INTO user_roles (user_id, role_ids, updated_at) VALUES ($1, $2, now()) ON CONFLICT (user_id) DO UPDATE SET role_ids = $2, updated_at = now()",
		userID, roleIDs,
	)
	return err
}

func (db *Database) LogUserMessage(ctx context.Context, userID string) error {
	_, err := db.Pool.Exec(ctx,
		"INSERT INTO personnel_stats (user_id, total_messages, last_message_at) VALUES ($1, 1, now()) ON CONFLICT (user_id) DO UPDATE SET total_messages = personnel_stats.total_messages + 1, last_message_at = now()",
		userID,
	)
	return err
}

func (db *Database) IncrementUserDeployments(ctx context.Context, userID string) error {
	_, err := db.Pool.Exec(ctx,
		"INSERT INTO personnel_stats (user_id, deployments_participated) VALUES ($1, 1) ON CONFLICT (user_id) DO UPDATE SET deployments_participated = personnel_stats.deployments_participated + 1",
		userID,
	)
	return err
}

type PersonnelStat struct {
	UserID                  string
	TotalMessages           int64
	DeploymentsParticipated int64
	LastMessageAt           *time.Time
}

func (db *Database) GetPersonnelStats(ctx context.Context) ([]PersonnelStat, error) {
	rows, err := db.Pool.Query(ctx, "SELECT user_id, total_messages, deployments_participated, last_message_at FROM personnel_stats ORDER BY total_messages DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var stats []PersonnelStat
	for rows.Next() {
		var s PersonnelStat
		if err := rows.Scan(&s.UserID, &s.TotalMessages, &s.DeploymentsParticipated, &s.LastMessageAt); err != nil {
			return nil, err
		}
		stats = append(stats, s)
	}
	return stats, nil
}

func (db *Database) GetUserRoles(ctx context.Context, userID string) ([]string, error) {
	var roleIDs []string
	err := db.Pool.QueryRow(ctx, "SELECT role_ids FROM user_roles WHERE user_id = $1", userID).Scan(&roleIDs)
	return roleIDs, err
}

func (db *Database) UpsertRole(ctx context.Context, roleID, name string) error {
	_, err := db.Pool.Exec(ctx,
		"INSERT INTO roles (id, name, updated_at) VALUES ($1, $2, now()) ON CONFLICT (id) DO UPDATE SET name = $2, updated_at = now()",
		roleID, name,
	)
	return err
}

type RoleConnection struct {
	ID      string `json:"id"`
	RoleIDA string `json:"roleIdA"`
	RoleIDB string `json:"roleIdB"`
}

func (db *Database) GetRoleConnections(ctx context.Context) ([]RoleConnection, error) {
	rows, err := db.Pool.Query(ctx, "SELECT id, role_id_a, role_id_b FROM role_connections")
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []RoleConnection
	for rows.Next() {
		var rc RoleConnection
		if err := rows.Scan(&rc.ID, &rc.RoleIDA, &rc.RoleIDB); err != nil {
			return nil, err
		}
		list = append(list, rc)
	}
	if list == nil {
		list = []RoleConnection{}
	}
	return list, nil
}

func (db *Database) AddRoleConnection(ctx context.Context, roleIDA, roleIDB string) (string, error) {
	var id string
	err := db.Pool.QueryRow(ctx, "INSERT INTO role_connections (role_id_a, role_id_b) VALUES ($1, $2) ON CONFLICT (role_id_a, role_id_b) DO UPDATE SET created_at = now() RETURNING id", roleIDA, roleIDB).Scan(&id)
	return id, err
}

func (db *Database) DeleteRoleConnection(ctx context.Context, id string) error {
	_, err := db.Pool.Exec(ctx, "DELETE FROM role_connections WHERE id = $1", id)
	return err
}

type Deployment struct {
	ID              string     `json:"id"`
	Message         string     `json:"message"`
	HostID          string     `json:"hostId"`
	CoHostID        string     `json:"coHostId"`
	Location        string     `json:"location"`
	Status          string     `json:"status"`
	DiscordMessageID string    `json:"discordMessageId"`
	StartedAt       *time.Time `json:"startedAt"`
	EndedAt         *time.Time `json:"endedAt"`
	DurationSeconds int64      `json:"durationSeconds"`
	AnnouncedBy     string     `json:"announcedBy"`
	CreatedAt       time.Time  `json:"createdAt"`
	UpdatedAt       time.Time  `json:"updatedAt"`
}

func (db *Database) CreateDeployment(ctx context.Context, message, hostID, coHostID, location, discordMessageID, announcedBy string) (string, error) {
	var id string
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO deployments (message, host_id, co_host_id, location, discord_message_id, announced_by)
		 VALUES ($1, $2, $3, $4, $5, $6) RETURNING id`,
		message, hostID, coHostID, location, discordMessageID, announcedBy,
	).Scan(&id)
	return id, err
}

func (db *Database) UpdateDeploymentStatus(ctx context.Context, discordMessageID, status string) error {
	_, err := db.Pool.Exec(ctx,
		"UPDATE deployments SET status = $1, updated_at = NOW() WHERE discord_message_id = $2",
		status, discordMessageID,
	)
	return err
}

func (db *Database) StartDeployment(ctx context.Context, discordMessageID string, startTime time.Time) error {
	_, err := db.Pool.Exec(ctx,
		"UPDATE deployments SET status = 'started', started_at = $1, updated_at = NOW() WHERE discord_message_id = $2",
		startTime, discordMessageID,
	)
	return err
}

func (db *Database) EndDeployment(ctx context.Context, discordMessageID string, endTime time.Time, durationSeconds int64) error {
	_, err := db.Pool.Exec(ctx,
		"UPDATE deployments SET status = 'ended', ended_at = $1, duration_seconds = $2, updated_at = NOW() WHERE discord_message_id = $3",
		endTime, durationSeconds, discordMessageID,
	)
	return err
}

func (db *Database) GetDeployments(ctx context.Context) ([]Deployment, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, message, host_id, co_host_id, location, status, discord_message_id,
		        started_at, ended_at, duration_seconds, announced_by, created_at, updated_at
		 FROM deployments ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []Deployment
	for rows.Next() {
		var d Deployment
		if err := rows.Scan(&d.ID, &d.Message, &d.HostID, &d.CoHostID, &d.Location,
			&d.Status, &d.DiscordMessageID, &d.StartedAt, &d.EndedAt,
			&d.DurationSeconds, &d.AnnouncedBy, &d.CreatedAt, &d.UpdatedAt); err != nil {
			return nil, err
		}
		list = append(list, d)
	}
	if list == nil {
		list = []Deployment{}
	}
	return list, nil
}

func (db *Database) GetDeploymentByMessageID(ctx context.Context, discordMessageID string) (*Deployment, error) {
	var d Deployment
	err := db.Pool.QueryRow(ctx,
		`SELECT id, message, host_id, co_host_id, location, status, discord_message_id,
		        started_at, ended_at, duration_seconds, announced_by, created_at, updated_at
		 FROM deployments WHERE discord_message_id = $1`,
		discordMessageID,
	).Scan(&d.ID, &d.Message, &d.HostID, &d.CoHostID, &d.Location,
		&d.Status, &d.DiscordMessageID, &d.StartedAt, &d.EndedAt,
		&d.DurationSeconds, &d.AnnouncedBy, &d.CreatedAt, &d.UpdatedAt)
	if err != nil {
		return nil, err
	}
	return &d, nil
}

func (db *Database) AddDeploymentParticipant(ctx context.Context, discordMessageID, userID string) error {
	_, err := db.Pool.Exec(ctx,
		`INSERT INTO deployment_participants (deployment_id, user_id)
		 SELECT id, $2 FROM deployments WHERE discord_message_id = $1
		 ON CONFLICT DO NOTHING`,
		discordMessageID, userID,
	)
	return err
}

func (db *Database) RemoveDeploymentParticipant(ctx context.Context, discordMessageID, userID string) error {
	_, err := db.Pool.Exec(ctx,
		`DELETE FROM deployment_participants
		 WHERE deployment_id = (SELECT id FROM deployments WHERE discord_message_id = $1)
		 AND user_id = $2`,
		discordMessageID, userID,
	)
	return err
}

func (db *Database) GetDeploymentParticipants(ctx context.Context, discordMessageID string) ([]string, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT user_id FROM deployment_participants
		 WHERE deployment_id = (SELECT id FROM deployments WHERE discord_message_id = $1)`,
		discordMessageID,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []string
	for rows.Next() {
		var uid string
		if err := rows.Scan(&uid); err != nil {
			return nil, err
		}
		list = append(list, uid)
	}
	return list, nil
}

func (db *Database) GetUserDeploymentCount(ctx context.Context, userID string) (int, error) {
	var count int
	err := db.Pool.QueryRow(ctx,
		`SELECT COUNT(*) FROM deployment_participants WHERE user_id = $1`,
		userID,
	).Scan(&count)
	return count, err
}

type AudioRecording struct {
	ID              string    `json:"id"`
	Name            string    `json:"name"`
	FileURL         string    `json:"fileUrl"`
	DurationSeconds float64   `json:"durationSeconds"`
	CreatedAt       time.Time `json:"createdAt"`
	UploadedBy      string    `json:"uploadedBy"`
}

func (db *Database) AddAudioRecording(ctx context.Context, name, fileURL string, durationSeconds float64, uploadedBy string) (string, error) {
	var id string
	err := db.Pool.QueryRow(ctx,
		`INSERT INTO audio_recordings (name, file_url, duration_seconds, uploaded_by) VALUES ($1, $2, $3, $4) RETURNING id`,
		name, fileURL, durationSeconds, uploadedBy,
	).Scan(&id)
	return id, err
}

func (db *Database) GetAudioRecordings(ctx context.Context) ([]AudioRecording, error) {
	rows, err := db.Pool.Query(ctx,
		`SELECT id, name, file_url, duration_seconds, created_at, uploaded_by FROM audio_recordings ORDER BY created_at DESC`,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()
	var list []AudioRecording
	for rows.Next() {
		var r AudioRecording
		if err := rows.Scan(&r.ID, &r.Name, &r.FileURL, &r.DurationSeconds, &r.CreatedAt, &r.UploadedBy); err != nil {
			return nil, err
		}
		list = append(list, r)
	}
	if list == nil {
		list = []AudioRecording{}
	}
	return list, nil
}

func (db *Database) DeleteAudioRecording(ctx context.Context, id string) error {
	_, err := db.Pool.Exec(ctx, `DELETE FROM audio_recordings WHERE id = $1`, id)
	return err
}

func (db *Database) GetAudioRecordingByID(ctx context.Context, id string) (*AudioRecording, error) {
	var r AudioRecording
	err := db.Pool.QueryRow(ctx,
		`SELECT id, name, file_url, duration_seconds, created_at, uploaded_by FROM audio_recordings WHERE id = $1`,
		id,
	).Scan(&r.ID, &r.Name, &r.FileURL, &r.DurationSeconds, &r.CreatedAt, &r.UploadedBy)
	if err != nil {
		return nil, err
	}
	return &r, nil
}
