package main

import (
	"database/sql"
	"encoding/json"
	"log"
	"os"
	"strings"
	"time"

	"github.com/google/uuid"
	_ "github.com/mattn/go-sqlite3"
	"golang.org/x/crypto/bcrypt"
)

// Types
type User struct {
	ID           string    `json:"id"`
	Email        string    `json:"email"`
	PasswordHash string    `json:"-"`
	MFAEnabled   bool      `json:"mfa_enabled"`
	MFASecret    string    `json:"-"`
	CreatedAt    time.Time `json:"created_at"`
}

type Project struct {
	ID                 string     `json:"id"`
	Name               string     `json:"name"`
	APIKey             string     `json:"api_key"`
	MaxEventsPerMonth  int        `json:"max_events_per_month"`
	CurrentMonthEvents int        `json:"current_month_events"`
	Coverage           float64    `json:"coverage"`
	CoverageUpdatedAt  *time.Time `json:"coverage_updated_at"`
	CreatedAt          time.Time  `json:"created_at"`
}

type ErrorEvent struct {
	ID          string    `json:"id"`
	ProjectID   string    `json:"project_id"`
	Message     string    `json:"message"`
	Level       string    `json:"level"`
	Environment string    `json:"environment"`
	Release     string    `json:"release"`
	Platform    string    `json:"platform"`
	Timestamp   time.Time `json:"timestamp"`
	Stacktrace  string    `json:"stacktrace"`
	Context     string    `json:"context"`
	User        string    `json:"user"`
	Tags        string    `json:"tags"`
	Status      string    `json:"status"`
	CreatedAt   time.Time `json:"created_at"`
}

type FileCoverage struct {
	ID         string  `json:"id"`
	SnapshotID string  `json:"snapshot_id"`
	FilePath   string  `json:"file_path"`
	Percentage float64 `json:"percentage"`
}

type Monitor struct {
	ID            string     `json:"id"`
	ProjectID     string     `json:"project_id"`
	Name          string     `json:"name"`
	Type          string     `json:"type"`
	URL           string     `json:"url"`
	Interval      int        `json:"interval"`
	Timeout       int        `json:"timeout"`
	Status        string     `json:"status"`
	LastCheckedAt *time.Time `json:"last_checked_at"`
	CreatedAt     time.Time  `json:"created_at"`
}

// Database initialization
func InitDB() (*sql.DB, error) {
	dbPath := os.Getenv("DB_PATH")
	if dbPath == "" {
		dbPath = "./data/sentry.db"
	}

	// Ensure data directory exists
	dataDir := "./data"
	if dbPath != "./data/sentry.db" {
		// If custom path, try to use its directory
		lastSlash := strings.LastIndex(dbPath, "/")
		if lastSlash != -1 {
			dataDir = dbPath[:lastSlash]
		}
	}

	if _, err := os.Stat(dataDir); os.IsNotExist(err) {
		_ = os.MkdirAll(dataDir, 0755)
	}

	db, err := sql.Open("sqlite3", dbPath)
	if err != nil {
		return nil, err
	}

	// Create tables
	projectsTable := `
	CREATE TABLE IF NOT EXISTS projects (
		id TEXT PRIMARY KEY,
		name TEXT NOT NULL,
		api_key TEXT UNIQUE NOT NULL,
		max_events_per_month INTEGER DEFAULT 1000,
		current_month_events INTEGER DEFAULT 0,
		coverage REAL DEFAULT 0,
		coverage_updated_at DATETIME,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	usersTable := `
	CREATE TABLE IF NOT EXISTS users (
		id TEXT PRIMARY KEY,
		email TEXT UNIQUE NOT NULL,
		password_hash TEXT NOT NULL,
		mfa_enabled BOOLEAN DEFAULT 0,
		mfa_secret TEXT DEFAULT '',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP
	);`

	errorsTable := `
	CREATE TABLE IF NOT EXISTS errors (
		id TEXT PRIMARY KEY,
		project_id TEXT NOT NULL,
		message TEXT NOT NULL,
		level TEXT,
		environment TEXT,
		release TEXT,
		platform TEXT,
		timestamp DATETIME,
		stacktrace TEXT,
		context TEXT,
		user TEXT,
		tags TEXT,
		status TEXT DEFAULT 'unresolved',
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY (project_id) REFERENCES projects(id)
	);`

	monitorsTable := `
	CREATE TABLE IF NOT EXISTS monitors (
		id TEXT PRIMARY KEY,
		project_id TEXT,
		name TEXT,
		type TEXT,
		url TEXT,
		interval INTEGER,
		timeout INTEGER DEFAULT 30,
		status TEXT,
		last_checked_at DATETIME,
		created_at DATETIME,
		FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
	);`

	projectSettingsTable := `
	CREATE TABLE IF NOT EXISTS project_settings (
		project_id TEXT PRIMARY KEY,
		notification_enabled BOOLEAN DEFAULT 1,
		notification_levels TEXT DEFAULT 'error,fatal',
		notification_frequency TEXT DEFAULT 'immediate',
		notification_email TEXT DEFAULT '',
		notification_webhook_url TEXT DEFAULT '',
		notification_rate_limit INTEGER DEFAULT 60,
		updated_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
	);`

	settingsTable := `
	CREATE TABLE IF NOT EXISTS settings (
		key TEXT PRIMARY KEY,
		value TEXT
	);`

	securityPoliciesTable := `
	CREATE TABLE IF NOT EXISTS security_policies (
		project_id TEXT PRIMARY KEY,
		ip_whitelist TEXT,
		allowed_domains TEXT,
		enforced BOOLEAN DEFAULT 0,
		FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
	);`

	coverageHistoryTable := `
	CREATE TABLE IF NOT EXISTS coverage_history (
		id TEXT PRIMARY KEY,
		project_id TEXT,
		percentage REAL,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
	);`

	fileCoverageSnapshotsTable := `
	CREATE TABLE IF NOT EXISTS file_coverage_snapshots (
		id TEXT PRIMARY KEY,
		snapshot_id TEXT,
		file_path TEXT,
		percentage REAL,
		FOREIGN KEY(snapshot_id) REFERENCES coverage_history(id) ON DELETE CASCADE
	);`

	apiKeyHistoryTable := `
	CREATE TABLE IF NOT EXISTS api_key_history (
		id TEXT PRIMARY KEY,
		project_id TEXT,
		api_key TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
	);`

	monitorChecksTable := `
	CREATE TABLE IF NOT EXISTS monitor_checks (
		id TEXT PRIMARY KEY,
		monitor_id TEXT,
		status TEXT,
		response_time INTEGER,
		status_code INTEGER,
		error_message TEXT,
		created_at DATETIME DEFAULT CURRENT_TIMESTAMP,
		FOREIGN KEY(monitor_id) REFERENCES monitors(id) ON DELETE CASCADE
	);`

	spansTable := `
	CREATE TABLE IF NOT EXISTS spans (
		id TEXT PRIMARY KEY,
		project_id TEXT,
		trace_id TEXT,
		span_id TEXT,
		parent_span_id TEXT,
		name TEXT,
		op TEXT,
		description TEXT,
		start_timestamp DATETIME,
		timestamp DATETIME,
		status TEXT,
		data TEXT,
		FOREIGN KEY(project_id) REFERENCES projects(id) ON DELETE CASCADE
	);`

	_, err = db.Exec(projectsTable)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(usersTable)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(errorsTable)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(monitorsTable)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(projectSettingsTable)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(settingsTable)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(securityPoliciesTable)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(coverageHistoryTable)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(fileCoverageSnapshotsTable)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(apiKeyHistoryTable)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(monitorChecksTable)
	if err != nil {
		return nil, err
	}

	_, err = db.Exec(spansTable)
	if err != nil {
		return nil, err
	}

	// Migrations
	db.Exec("ALTER TABLE errors ADD COLUMN status TEXT DEFAULT 'unresolved';")
	db.Exec("ALTER TABLE projects ADD COLUMN max_events_per_month INTEGER DEFAULT 1000;")
	db.Exec("ALTER TABLE projects ADD COLUMN current_month_events INTEGER DEFAULT 0;")
	db.Exec("ALTER TABLE users ADD COLUMN mfa_enabled BOOLEAN DEFAULT 0;")
	db.Exec("ALTER TABLE users ADD COLUMN mfa_secret TEXT DEFAULT '';")
	db.Exec("ALTER TABLE projects ADD COLUMN coverage REAL DEFAULT 0;")
	db.Exec("ALTER TABLE projects ADD COLUMN coverage_updated_at DATETIME;")
	db.Exec("ALTER TABLE monitors ADD COLUMN timeout INTEGER DEFAULT 30;")

	// Seed admin user
	if err := seedAdminUser(db); err != nil {
		log.Printf("Warning: Failed to seed admin user: %v", err)
	}

	log.Println("Database initialized successfully")
	return db, nil
}

func seedAdminUser(db *sql.DB) error {
	var count int
	err := db.QueryRow("SELECT COUNT(*) FROM users").Scan(&count)
	if err != nil {
		return err
	}

	if count > 0 {
		return nil
	}

	email := os.Getenv("ADMIN_EMAIL")
	password := os.Getenv("ADMIN_PASSWORD")

	if email == "" || password == "" {
		log.Println("ADMIN_EMAIL or ADMIN_PASSWORD not set, skipping admin seeding")
		return nil
	}

	hashedPassword, err := bcrypt.GenerateFromPassword([]byte(password), bcrypt.DefaultCost)
	if err != nil {
		return err
	}

	id := uuid.New().String()
	_, err = db.Exec("INSERT INTO users (id, email, password_hash) VALUES (?, ?, ?)", id, email, string(hashedPassword))
	if err != nil {
		return err
	}

	log.Printf("Seeded admin user: %s", email)
	return nil
}

// User functions
func GetUserByEmail(db *sql.DB, email string) (*User, error) {
	var u User
	err := db.QueryRow("SELECT id, email, password_hash, mfa_enabled, mfa_secret, created_at FROM users WHERE email = ?", email).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.MFAEnabled, &u.MFASecret, &u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func GetUserByID(db *sql.DB, id string) (*User, error) {
	var u User
	err := db.QueryRow("SELECT id, email, password_hash, mfa_enabled, mfa_secret, created_at FROM users WHERE id = ?", id).Scan(
		&u.ID, &u.Email, &u.PasswordHash, &u.MFAEnabled, &u.MFASecret, &u.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &u, nil
}

func UpdateUserMFA(db *sql.DB, userID string, enabled bool, secret string) error {
	_, err := db.Exec("UPDATE users SET mfa_enabled = ?, mfa_secret = ? WHERE id = ?", enabled, secret, userID)
	return err
}

// Project functions
func GetProjectByAPIKey(db *sql.DB, apiKey string) (*Project, error) {
	var project Project
	err := db.QueryRow(
		"SELECT id, name, api_key, max_events_per_month, current_month_events, created_at FROM projects WHERE api_key = ?",
		apiKey,
	).Scan(&project.ID, &project.Name, &project.APIKey, &project.MaxEventsPerMonth, &project.CurrentMonthEvents, &project.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &project, nil
}

func GetAllProjects(db *sql.DB) ([]Project, error) {
	rows, err := db.Query("SELECT id, name, api_key, max_events_per_month, current_month_events, coverage, coverage_updated_at, created_at FROM projects ORDER BY created_at DESC")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var projects []Project
	for rows.Next() {
		var p Project
		err := rows.Scan(&p.ID, &p.Name, &p.APIKey, &p.MaxEventsPerMonth, &p.CurrentMonthEvents, &p.Coverage, &p.CoverageUpdatedAt, &p.CreatedAt)
		if err != nil {
			return nil, err
		}
		projects = append(projects, p)
	}
	return projects, nil
}

func GetProject(db *sql.DB, id string) (*Project, error) {
	var p Project
	err := db.QueryRow(
		"SELECT id, name, api_key, max_events_per_month, current_month_events, coverage, coverage_updated_at, created_at FROM projects WHERE id = ?",
		id,
	).Scan(&p.ID, &p.Name, &p.APIKey, &p.MaxEventsPerMonth, &p.CurrentMonthEvents, &p.Coverage, &p.CoverageUpdatedAt, &p.CreatedAt)
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func CreateProject(db *sql.DB, name string) (*Project, error) {
	id := uuid.New().String()
	apiKey := uuid.New().String()

	_, err := db.Exec(
		"INSERT INTO projects (id, name, api_key) VALUES (?, ?, ?)",
		id, name, apiKey,
	)
	if err != nil {
		return nil, err
	}

	return &Project{
		ID:                 id,
		Name:               name,
		APIKey:             apiKey,
		MaxEventsPerMonth:  1000,
		CurrentMonthEvents: 0,
		CreatedAt:          time.Now(),
	}, nil
}

func IncrementProjectEventCount(db *sql.DB, projectID string) error {
	_, err := db.Exec("UPDATE projects SET current_month_events = current_month_events + 1 WHERE id = ?", projectID)
	return err
}

func UpdateProjectQuota(db *sql.DB, id string, quota int) error {
	_, err := db.Exec("UPDATE projects SET max_events_per_month = ? WHERE id = ?", quota, id)
	return err
}

func UpdateProjectCoverage(db *sql.DB, id string, coverage float64, files []FileCoverage) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec("UPDATE projects SET coverage = ?, coverage_updated_at = ? WHERE id = ?", coverage, time.Now(), id)
	if err != nil {
		return err
	}

	// Insert into coverage history
	snapshotID := uuid.New().String()
	_, err = tx.Exec("INSERT INTO coverage_history (id, project_id, percentage) VALUES (?, ?, ?)", snapshotID, id, coverage)
	if err != nil {
		return err
	}

	// Insert file snapshots if available
	if len(files) > 0 {
		for _, f := range files {
			fID := uuid.New().String()
			_, err = tx.Exec("INSERT INTO file_coverage_snapshots (id, snapshot_id, file_path, percentage) VALUES (?, ?, ?, ?)", fID, snapshotID, f.FilePath, f.Percentage)
			if err != nil {
				return err
			}
		}
	}

	return tx.Commit()
}

func GetProjectFileCoverage(db *sql.DB, snapshotID string) ([]FileCoverage, error) {
	rows, err := db.Query("SELECT id, snapshot_id, file_path, percentage FROM file_coverage_snapshots WHERE snapshot_id = ?", snapshotID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var files []FileCoverage
	for rows.Next() {
		var f FileCoverage
		if err := rows.Scan(&f.ID, &f.SnapshotID, &f.FilePath, &f.Percentage); err != nil {
			return nil, err
		}
		files = append(files, f)
	}
	return files, nil
}

func DeleteProject(db *sql.DB, id string) error {
	_, err := db.Exec("DELETE FROM projects WHERE id = ?", id)
	return err
}

// Error functions
func InsertError(db *sql.DB, event *ErrorEvent) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO errors (id, project_id, message, level, environment, release, platform, timestamp, stacktrace, context, user, tags, status, created_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		event.ID, event.ProjectID, event.Message, event.Level, event.Environment,
		event.Release, event.Platform, event.Timestamp, event.Stacktrace, event.Context,
		event.User, event.Tags, event.Status, event.CreatedAt,
	)
	if err != nil {
		return err
	}

	_, err = tx.Exec("UPDATE projects SET current_month_events = current_month_events + 1 WHERE id = ?", event.ProjectID)
	if err != nil {
		return err
	}

	return tx.Commit()
}

func GetErrors(db *sql.DB, projectID string, limit, offset int, status string) ([]ErrorEvent, int, error) {
	baseQuery := "FROM errors WHERE project_id = ?"
	args := []interface{}{projectID}

	if status != "" {
		baseQuery += " AND status = ?"
		args = append(args, status)
	}

	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	selectQuery := `SELECT id, project_id, message, level, environment, release, platform, timestamp, stacktrace, context, user, tags, status, created_at ` + baseQuery + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.Query(selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var errors []ErrorEvent
	for rows.Next() {
		var e ErrorEvent
		err := rows.Scan(
			&e.ID, &e.ProjectID, &e.Message, &e.Level, &e.Environment,
			&e.Release, &e.Platform, &e.Timestamp, &e.Stacktrace, &e.Context,
			&e.User, &e.Tags, &e.Status, &e.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		errors = append(errors, e)
	}
	return errors, total, nil
}

func GetError(db *sql.DB, id string) (*ErrorEvent, error) {
	var e ErrorEvent
	err := db.QueryRow(
		`SELECT id, project_id, message, level, environment, release, platform, timestamp, stacktrace, context, user, tags, status, created_at
		 FROM errors WHERE id = ?`,
		id,
	).Scan(
		&e.ID, &e.ProjectID, &e.Message, &e.Level, &e.Environment,
		&e.Release, &e.Platform, &e.Timestamp, &e.Stacktrace, &e.Context,
		&e.User, &e.Tags, &e.Status, &e.CreatedAt,
	)
	if err != nil {
		return nil, err
	}
	return &e, nil
}

func DeleteError(db *sql.DB, id string) error {
	_, err := db.Exec("DELETE FROM errors WHERE id = ?", id)
	return err
}

func UpdateErrorStatus(db *sql.DB, id string, status string) error {
	_, err := db.Exec("UPDATE errors SET status = ? WHERE id = ?", status, id)
	return err
}

func GetErrorsWithStats(db *sql.DB, projectID string, limit, offset int, status string) ([]map[string]interface{}, int, error) {
	baseQuery := "FROM errors WHERE project_id = ?"
	args := []interface{}{projectID}

	if status != "" {
		baseQuery += " AND status = ?"
		args = append(args, status)
	}

	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	selectQuery := `SELECT id, project_id, message, level, environment, release, platform, timestamp, stacktrace, context, user, tags, status, created_at ` + baseQuery + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.Query(selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var errors []ErrorEvent
	for rows.Next() {
		var e ErrorEvent
		err := rows.Scan(
			&e.ID, &e.ProjectID, &e.Message, &e.Level, &e.Environment,
			&e.Release, &e.Platform, &e.Timestamp, &e.Stacktrace, &e.Context,
			&e.User, &e.Tags, &e.Status, &e.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		errors = append(errors, e)
	}

	result := make([]map[string]interface{}, 0, len(errors))
	for _, e := range errors {
		var eventCount int
		countQuery := "SELECT COUNT(*) FROM errors WHERE message = ? AND project_id = ?"
		if status != "" {
			countQuery += " AND status = ?"
			db.QueryRow(countQuery, e.Message, projectID, status).Scan(&eventCount)
		} else {
			db.QueryRow(countQuery, e.Message, projectID).Scan(&eventCount)
		}

		userRows, _ := db.Query("SELECT user FROM errors WHERE message = ? AND project_id = ? AND user IS NOT NULL AND user != ''", e.Message, projectID)
		userSet := make(map[string]bool)
		for userRows.Next() {
			var userJSON string
			if err := userRows.Scan(&userJSON); err != nil {
				continue
			}
			var userData map[string]interface{}
			if err := json.Unmarshal([]byte(userJSON), &userData); err != nil {
				continue
			}
			var userID string
			if id, ok := userData["id"].(string); ok && id != "" {
				userID = id
			} else if email, ok := userData["email"].(string); ok && email != "" {
				userID = email
			} else if username, ok := userData["username"].(string); ok && username != "" {
				userID = username
			} else {
				userID = userJSON
			}
			if userID != "" {
				userSet[userID] = true
			}
		}
		userRows.Close()
		userCount := len(userSet)

		errorMap := map[string]interface{}{
			"id":          e.ID,
			"project_id":  e.ProjectID,
			"message":     e.Message,
			"level":       e.Level,
			"environment": e.Environment,
			"release":     e.Release,
			"platform":    e.Platform,
			"timestamp":   e.Timestamp,
			"stacktrace":  e.Stacktrace,
			"context":     e.Context,
			"user":        e.User,
			"tags":        e.Tags,
			"status":      e.Status,
			"created_at":  e.CreatedAt,
			"event_count": eventCount,
			"user_count":  userCount,
		}
		result = append(result, errorMap)
	}
	return result, total, nil
}

// Trace functions
type TraceSpan struct {
	ID             string    `json:"id"`
	ProjectID      string    `json:"project_id"`
	TraceID        string    `json:"trace_id"`
	SpanID         string    `json:"span_id"`
	ParentSpanID   string    `json:"parent_span_id,omitempty"`
	Name           string    `json:"name"`
	Op             string    `json:"op,omitempty"`
	Description    string    `json:"description,omitempty"`
	StartTimestamp time.Time `json:"start_timestamp"`
	Timestamp      time.Time `json:"timestamp"`
	Status         string    `json:"status,omitempty"`
	Data           string    `json:"data,omitempty"`
}

func InsertSpan(db *sql.DB, span *TraceSpan) error {
	tx, err := db.Begin()
	if err != nil {
		return err
	}
	defer tx.Rollback()

	_, err = tx.Exec(
		`INSERT INTO spans (id, project_id, trace_id, span_id, parent_span_id, name, op, description, start_timestamp, timestamp, status, data)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		span.ID, span.ProjectID, span.TraceID, span.SpanID, span.ParentSpanID,
		span.Name, span.Op, span.Description, span.StartTimestamp, span.Timestamp,
		span.Status, span.Data,
	)
	if err != nil {
		return err
	}

	// Only count root spans (transactions) as events for quota/stats
	if span.ParentSpanID == "" {
		_, err = tx.Exec("UPDATE projects SET current_month_events = current_month_events + 1 WHERE id = ?", span.ProjectID)
		if err != nil {
			return err
		}
	}

	return tx.Commit()
}

type HourlyStat struct {
	Hour   string `json:"hour"`
	Errors int    `json:"errors"`
	Traces int    `json:"traces"`
}

func GetHourlyStats(db *sql.DB, projectID string) ([]HourlyStat, error) {
	// Initialize 24h of data
	stats := make([]HourlyStat, 24)
	now := time.Now()
	for i := 0; i < 24; i++ {
		t := now.Add(time.Duration(-(23 - i)) * time.Hour)
		stats[i] = HourlyStat{
			Hour:   t.Format("2006-01-02 15:00:00"),
			Errors: 0,
			Traces: 0,
		}
	}

	// Fetch Errors
	errorQuery := `SELECT strftime('%Y-%m-%d %H:00:00', timestamp) as h, count(*)
				   FROM errors
				   WHERE timestamp > datetime('now', '-24 hours') `
	args := []interface{}{}
	if projectID != "" {
		errorQuery += " AND project_id = ? "
		args = append(args, projectID)
	}
	errorQuery += " GROUP BY h"

	rows, err := db.Query(errorQuery, args...)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var h string
			var count int
			if err := rows.Scan(&h, &count); err == nil {
				for i := range stats {
					if stats[i].Hour == h {
						stats[i].Errors = count
						break
					}
				}
			}
		}
	}

	// Fetch Traces (Root Spans)
	traceQuery := `SELECT strftime('%Y-%m-%d %H:00:00', start_timestamp) as h, count(*)
				   FROM spans
				   WHERE start_timestamp > datetime('now', '-24 hours') AND parent_span_id = '' `
	args = []interface{}{}
	if projectID != "" {
		traceQuery += " AND project_id = ? "
		args = append(args, projectID)
	}
	traceQuery += " GROUP BY h"

	rows, err = db.Query(traceQuery, args...)
	if err == nil {
		defer rows.Close()
		for rows.Next() {
			var h string
			var count int
			if err := rows.Scan(&h, &count); err == nil {
				for i := range stats {
					if stats[i].Hour == h {
						stats[i].Traces = count
						break
					}
				}
			}
		}
	}

	return stats, nil
}

func GetProjectRootSpans(db *sql.DB, projectID string, limit int) ([]TraceSpan, error) {
	rows, err := db.Query(`
		SELECT id, project_id, trace_id, span_id, parent_span_id,
		       name, op, description, start_timestamp, timestamp, status, data
		FROM spans
		WHERE project_id = ? AND (parent_span_id IS NULL OR parent_span_id = '')
		ORDER BY start_timestamp DESC LIMIT ?`, projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spans []TraceSpan
	for rows.Next() {
		var s TraceSpan
		err := rows.Scan(
			&s.ID, &s.ProjectID, &s.TraceID, &s.SpanID, &s.ParentSpanID,
			&s.Name, &s.Op, &s.Description, &s.StartTimestamp, &s.Timestamp, &s.Status, &s.Data,
		)
		if err != nil {
			return nil, err
		}
		spans = append(spans, s)
	}
	return spans, nil
}

func GetTraceSpans(db *sql.DB, traceID string) ([]TraceSpan, error) {
	rows, err := db.Query(`
		SELECT id, project_id, trace_id, span_id, parent_span_id,
		       name, op, description, start_timestamp, timestamp, status, data
		FROM spans WHERE trace_id = ? ORDER BY start_timestamp ASC`, traceID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var spans []TraceSpan
	for rows.Next() {
		var s TraceSpan
		err := rows.Scan(
			&s.ID, &s.ProjectID, &s.TraceID, &s.SpanID, &s.ParentSpanID,
			&s.Name, &s.Op, &s.Description, &s.StartTimestamp, &s.Timestamp, &s.Status, &s.Data,
		)
		if err != nil {
			return nil, err
		}
		spans = append(spans, s)
	}
	return spans, nil
}

// Monitor functions
func GetAllActiveMonitors(db *sql.DB) ([]Monitor, error) {
	rows, err := db.Query("SELECT id, project_id, name, type, url, interval, timeout, status, last_checked_at, created_at FROM monitors WHERE status != 'paused'")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var monitors []Monitor
	for rows.Next() {
		var m Monitor
		var lastChecked sql.NullTime
		var timeout sql.NullInt64
		err := rows.Scan(&m.ID, &m.ProjectID, &m.Name, &m.Type, &m.URL, &m.Interval, &timeout, &m.Status, &lastChecked, &m.CreatedAt)
		if err != nil {
			return nil, err
		}
		if lastChecked.Valid {
			m.LastCheckedAt = &lastChecked.Time
		}
		if timeout.Valid {
			m.Timeout = int(timeout.Int64)
		} else {
			m.Timeout = 30
		}
		monitors = append(monitors, m)
	}
	return monitors, nil
}

func GetProjectMonitors(db *sql.DB, projectID string) ([]Monitor, error) {
	rows, err := db.Query("SELECT id, project_id, name, type, url, interval, timeout, status, last_checked_at, created_at FROM monitors WHERE project_id = ? ORDER BY created_at DESC", projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var monitors []Monitor
	for rows.Next() {
		var m Monitor
		var lastChecked sql.NullTime
		var timeout sql.NullInt64
		err := rows.Scan(&m.ID, &m.ProjectID, &m.Name, &m.Type, &m.URL, &m.Interval, &timeout, &m.Status, &lastChecked, &m.CreatedAt)
		if err != nil {
			return nil, err
		}
		if lastChecked.Valid {
			m.LastCheckedAt = &lastChecked.Time
		}
		if timeout.Valid {
			m.Timeout = int(timeout.Int64)
		} else {
			m.Timeout = 30
		}
		monitors = append(monitors, m)
	}
	return monitors, nil
}

func CreateMonitor(db *sql.DB, monitor *Monitor) error {
	_, err := db.Exec(`
		INSERT INTO monitors (id, project_id, name, type, url, interval, timeout, status, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?)`,
		monitor.ID, monitor.ProjectID, monitor.Name, monitor.Type, monitor.URL, monitor.Interval, monitor.Timeout, monitor.Status, monitor.CreatedAt,
	)
	return err
}

type MonitorCheck struct {
	ID           string    `json:"id"`
	MonitorID    string    `json:"monitor_id"`
	Status       string    `json:"status"`
	ResponseTime int       `json:"response_time"`
	StatusCode   int       `json:"status_code"`
	ErrorMessage string    `json:"error_message"`
	CreatedAt    time.Time `json:"created_at"`
}

func InsertMonitorCheck(db *sql.DB, check *MonitorCheck) error {
	_, err := db.Exec(`
		INSERT INTO monitor_checks (id, monitor_id, status, response_time, status_code, error_message, created_at)
		VALUES (?, ?, ?, ?, ?, ?, ?)`,
		check.ID, check.MonitorID, check.Status, check.ResponseTime, check.StatusCode, check.ErrorMessage, check.CreatedAt,
	)
	if err != nil {
		return err
	}
	_, err = db.Exec("UPDATE monitors SET status = ?, last_checked_at = ? WHERE id = ?", check.Status, check.CreatedAt, check.MonitorID)
	return err
}

func GetMonitorChecks(db *sql.DB, monitorID string, limit int) ([]MonitorCheck, error) {
	rows, err := db.Query(
		"SELECT id, monitor_id, status, response_time, status_code, error_message, created_at FROM monitor_checks WHERE monitor_id = ? ORDER BY created_at DESC LIMIT ?",
		monitorID, limit,
	)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var checks []MonitorCheck
	for rows.Next() {
		var c MonitorCheck
		err := rows.Scan(&c.ID, &c.MonitorID, &c.Status, &c.ResponseTime, &c.StatusCode, &c.ErrorMessage, &c.CreatedAt)
		if err != nil {
			return nil, err
		}
		checks = append(checks, c)
	}
	return checks, nil
}

// Settings functions
type SecurityPolicy struct {
	ProjectID      string `json:"project_id"`
	IPWhitelist    string `json:"ip_whitelist"`
	AllowedDomains string `json:"allowed_domains"`
	Enforced       bool   `json:"enforced"`
}

func GetSecurityPolicy(db *sql.DB, projectID string) (*SecurityPolicy, error) {
	var p SecurityPolicy
	err := db.QueryRow(
		"SELECT project_id, ip_whitelist, allowed_domains, enforced FROM security_policies WHERE project_id = ?",
		projectID,
	).Scan(&p.ProjectID, &p.IPWhitelist, &p.AllowedDomains, &p.Enforced)

	if err == sql.ErrNoRows {
		return &SecurityPolicy{ProjectID: projectID, Enforced: false}, nil
	}
	if err != nil {
		return nil, err
	}
	return &p, nil
}

func GetSetting(db *sql.DB, key string) (string, error) {
	var value string
	err := db.QueryRow("SELECT value FROM settings WHERE key = ?", key).Scan(&value)
	if err == sql.ErrNoRows {
		return "", nil
	}
	return value, err
}

func UpdateSetting(db *sql.DB, key, value string) error {
	_, err := db.Exec("INSERT OR REPLACE INTO settings (key, value) VALUES (?, ?)", key, value)
	return err
}

func GetAllSettings(db *sql.DB) (map[string]string, error) {
	rows, err := db.Query("SELECT key, value FROM settings")
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	settings := make(map[string]string)
	for rows.Next() {
		var key, value string
		if err := rows.Scan(&key, &value); err != nil {
			return nil, err
		}
		settings[key] = value
	}
	return settings, nil
}

func UpdateSecurityPolicy(db *sql.DB, p *SecurityPolicy) error {
	_, err := db.Exec(
		"INSERT OR REPLACE INTO security_policies (project_id, ip_whitelist, allowed_domains, enforced) VALUES (?, ?, ?, ?)",
		p.ProjectID, p.IPWhitelist, p.AllowedDomains, p.Enforced,
	)
	return err
}

func GetAllErrorsWithStats(db *sql.DB, limit, offset int, status string) ([]map[string]interface{}, int, error) {
	baseQuery := "FROM errors"
	var args []interface{}

	if status != "" {
		baseQuery += " WHERE status = ?"
		args = append(args, status)
	}

	var total int
	countQuery := "SELECT COUNT(*) " + baseQuery
	err := db.QueryRow(countQuery, args...).Scan(&total)
	if err != nil {
		return nil, 0, err
	}

	selectQuery := `SELECT id, project_id, message, level, environment, release, platform, timestamp, stacktrace, context, user, tags, status, created_at ` + baseQuery + " ORDER BY created_at DESC LIMIT ? OFFSET ?"
	args = append(args, limit, offset)

	rows, err := db.Query(selectQuery, args...)
	if err != nil {
		return nil, 0, err
	}
	defer rows.Close()

	var errors []ErrorEvent
	for rows.Next() {
		var e ErrorEvent
		err := rows.Scan(
			&e.ID, &e.ProjectID, &e.Message, &e.Level, &e.Environment,
			&e.Release, &e.Platform, &e.Timestamp, &e.Stacktrace, &e.Context,
			&e.User, &e.Tags, &e.Status, &e.CreatedAt,
		)
		if err != nil {
			return nil, 0, err
		}
		errors = append(errors, e)
	}

	result := make([]map[string]interface{}, 0, len(errors))
	for _, e := range errors {
		var eventCount int
		countQuery := "SELECT COUNT(*) FROM errors WHERE message = ?"
		if status != "" {
			countQuery += " AND status = ?"
			db.QueryRow(countQuery, e.Message, status).Scan(&eventCount)
		} else {
			db.QueryRow(countQuery, e.Message).Scan(&eventCount)
		}

		userRows, _ := db.Query("SELECT user FROM errors WHERE message = ? AND user IS NOT NULL AND user != ''", e.Message)
		userSet := make(map[string]bool)
		for userRows.Next() {
			var userJSON string
			if err := userRows.Scan(&userJSON); err != nil {
				continue
			}
			var userData map[string]interface{}
			if err := json.Unmarshal([]byte(userJSON), &userData); err != nil {
				continue
			}
			var userID string
			if id, ok := userData["id"].(string); ok && id != "" {
				userID = id
			} else if email, ok := userData["email"].(string); ok && email != "" {
				userID = email
			} else if username, ok := userData["username"].(string); ok && username != "" {
				userID = username
			} else {
				userID = userJSON
			}
			if userID != "" {
				userSet[userID] = true
			}
		}
		userRows.Close()
		userCount := len(userSet)

		errorMap := map[string]interface{}{
			"id":          e.ID,
			"project_id":  e.ProjectID,
			"message":     e.Message,
			"level":       e.Level,
			"environment": e.Environment,
			"release":     e.Release,
			"platform":    e.Platform,
			"timestamp":   e.Timestamp,
			"stacktrace":  e.Stacktrace,
			"context":     e.Context,
			"user":        e.User,
			"tags":        e.Tags,
			"status":      e.Status,
			"created_at":  e.CreatedAt,
			"event_count": eventCount,
			"user_count":  userCount,
		}
		result = append(result, errorMap)
	}
	return result, total, nil
}

type CoverageSnapshot struct {
	ID         string    `json:"id"`
	ProjectID  string    `json:"project_id"`
	Percentage float64   `json:"percentage"`
	CreatedAt  time.Time `json:"created_at"`
}

func GetProjectCoverageHistory(db *sql.DB, projectID string, limit int) ([]CoverageSnapshot, error) {
	rows, err := db.Query("SELECT id, project_id, percentage, created_at FROM coverage_history WHERE project_id = ? ORDER BY created_at DESC LIMIT ?", projectID, limit)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []CoverageSnapshot
	for rows.Next() {
		var s CoverageSnapshot
		if err := rows.Scan(&s.ID, &s.ProjectID, &s.Percentage, &s.CreatedAt); err != nil {
			return nil, err
		}
		history = append(history, s)
	}
	return history, nil
}

type APIKeyHistory struct {
	ID        string    `json:"id"`
	ProjectID string    `json:"project_id"`
	APIKey    string    `json:"api_key"`
	CreatedAt time.Time `json:"created_at"`
}

func RotateAPIKey(db *sql.DB, projectID string) (string, error) {
	var currentKey string
	err := db.QueryRow("SELECT api_key FROM projects WHERE id = ?", projectID).Scan(&currentKey)
	if err != nil {
		return "", err
	}

	archiveID := uuid.New().String()
	_, err = db.Exec("INSERT INTO api_key_history (id, project_id, api_key) VALUES (?, ?, ?)", archiveID, projectID, currentKey)
	if err != nil {
		return "", err
	}

	newKey := uuid.New().String()
	_, err = db.Exec("UPDATE projects SET api_key = ? WHERE id = ?", newKey, projectID)
	if err != nil {
		return "", err
	}

	return newKey, nil
}

func GetAPIKeyHistory(db *sql.DB, projectID string) ([]APIKeyHistory, error) {
	rows, err := db.Query("SELECT id, project_id, api_key, created_at FROM api_key_history WHERE project_id = ? ORDER BY created_at DESC", projectID)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	var history []APIKeyHistory
	for rows.Next() {
		var h APIKeyHistory
		if err := rows.Scan(&h.ID, &h.ProjectID, &h.APIKey, &h.CreatedAt); err != nil {
			return nil, err
		}
		history = append(history, h)
	}
	return history, nil
}

// Project Settings

type ProjectSettings struct {
	ProjectID              string    `json:"project_id"`
	NotificationEnabled    bool      `json:"notification_enabled"`
	NotificationLevels     string    `json:"notification_levels"`    // comma-separated: error,fatal,warning,info
	NotificationFrequency  string    `json:"notification_frequency"` // immediate, hourly, daily
	NotificationEmail      string    `json:"notification_email"`
	NotificationWebhookURL string    `json:"notification_webhook_url"`
	NotificationRateLimit  int       `json:"notification_rate_limit"` // minutes
	UpdatedAt              time.Time `json:"updated_at"`
}

func GetProjectSettings(db *sql.DB, projectID string) (*ProjectSettings, error) {
	var s ProjectSettings
	err := db.QueryRow(
		`SELECT project_id, notification_enabled, notification_levels, notification_frequency,
		 notification_email, notification_webhook_url, notification_rate_limit, updated_at
		 FROM project_settings WHERE project_id = ?`,
		projectID,
	).Scan(
		&s.ProjectID, &s.NotificationEnabled, &s.NotificationLevels, &s.NotificationFrequency,
		&s.NotificationEmail, &s.NotificationWebhookURL, &s.NotificationRateLimit, &s.UpdatedAt,
	)

	if err == sql.ErrNoRows {
		// Return default settings if none exist
		return &ProjectSettings{
			ProjectID:              projectID,
			NotificationEnabled:    true,
			NotificationLevels:     "error,fatal",
			NotificationFrequency:  "immediate",
			NotificationEmail:      "",
			NotificationWebhookURL: "",
			NotificationRateLimit:  60,
			UpdatedAt:              time.Now(),
		}, nil
	}
	if err != nil {
		return nil, err
	}
	return &s, nil
}

func UpdateProjectSettings(db *sql.DB, s *ProjectSettings) error {
	_, err := db.Exec(
		`INSERT OR REPLACE INTO project_settings
		 (project_id, notification_enabled, notification_levels, notification_frequency,
		  notification_email, notification_webhook_url, notification_rate_limit, updated_at)
		 VALUES (?, ?, ?, ?, ?, ?, ?, ?)`,
		s.ProjectID, s.NotificationEnabled, s.NotificationLevels, s.NotificationFrequency,
		s.NotificationEmail, s.NotificationWebhookURL, s.NotificationRateLimit, time.Now(),
	)
	return err
}

func UpdateProjectName(db *sql.DB, projectID, name string) error {
	_, err := db.Exec("UPDATE projects SET name = ? WHERE id = ?", name, projectID)
	return err
}
