package main

import (
	"crypto/sha256"
	"database/sql"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

// GetErrorGroups returns grouped errors with aggregated stats
func GetErrorGroups(db *sql.DB, projectID string, limit int, cursor string, status string) ([]map[string]interface{}, string, bool, error) {
	// Build the base query for grouped errors
	baseQuery := `
		SELECT 
			fingerprint,
			message,
			level,
			status,
			project_id,
			MIN(created_at) as first_seen,
			MAX(created_at) as last_seen,
			COUNT(*) as event_count,
			environment,
			platform,
			MAX(id) as representative_id
		FROM errors
		WHERE project_id = ?`

	args := []interface{}{projectID}

	if status != "" {
		baseQuery += " AND status = ?"
		args = append(args, status)
	}

	// Cursor-based pagination on last_seen
	if cursor != "" {
		baseQuery += " AND MAX(created_at) < ?"
		args = append(args, cursor)
	}

	baseQuery += " GROUP BY fingerprint, message, level, status, project_id, environment, platform"
	baseQuery += " ORDER BY MAX(created_at) DESC LIMIT ?"
	args = append(args, limit+1)

	rows, err := db.Query(baseQuery, args...)
	if err != nil {
		return nil, "", false, err
	}
	defer rows.Close()

	var groups []ErrorGroup
	for rows.Next() {
		var g ErrorGroup
		err := rows.Scan(
			&g.Fingerprint, &g.Message, &g.Level, &g.Status, &g.ProjectID,
			&g.FirstSeen, &g.LastSeen, &g.EventCount,
			&g.Environment, &g.Platform, &g.RepresentativeID,
		)
		if err != nil {
			return nil, "", false, err
		}
		groups = append(groups, g)
	}

	hasMore := len(groups) > limit
	if hasMore {
		groups = groups[:limit]
	}

	// Get user counts for each group
	fingerprints := make([]string, len(groups))
	for i, g := range groups {
		fingerprints[i] = g.Fingerprint
	}

	userCounts, err := getUserCountsByFingerprint(db, projectID, fingerprints)
	if err != nil {
		// Log but don't fail the request
		fmt.Printf("Warning: failed to get user counts: %v\n", err)
		userCounts = make(map[string]int)
	}

	// Get timeline data for each group
	timelines, err := getTimelinesByFingerprint(db, projectID, fingerprints, 24*time.Hour)
	if err != nil {
		fmt.Printf("Warning: failed to get timelines: %v\n", err)
		timelines = make(map[string][]TimelinePoint)
	}

	// Build result with all metadata
	result := make([]map[string]interface{}, 0, len(groups))
	for _, g := range groups {
		groupMap := map[string]interface{}{
			"id":                g.RepresentativeID, // Use representative ID for routing
			"fingerprint":       g.Fingerprint,
			"message":           g.Message,
			"level":             g.Level,
			"status":            g.Status,
			"project_id":        g.ProjectID,
			"first_seen":        g.FirstSeen,
			"last_seen":         g.LastSeen,
			"event_count":       g.EventCount,
			"user_count":        userCounts[g.Fingerprint],
			"environment":       g.Environment,
			"platform":          g.Platform,
			"timeline":          timelines[g.Fingerprint],
			"representative_id": g.RepresentativeID,
		}
		result = append(result, groupMap)
	}

	nextCursor := ""
	if len(groups) > 0 {
		nextCursor = groups[len(groups)-1].LastSeen.Format(time.RFC3339Nano)
	}

	return result, nextCursor, hasMore, nil
}

// TimelinePoint represents a point in the error timeline
type TimelinePoint struct {
	Timestamp time.Time `json:"timestamp"`
	Count     int       `json:"count"`
}

func getUserCountsByFingerprint(db *sql.DB, projectID string, fingerprints []string) (map[string]int, error) {
	if len(fingerprints) == 0 {
		return make(map[string]int), nil
	}

	placeholders := strings.Repeat("?,", len(fingerprints))
	placeholders = placeholders[:len(placeholders)-1]

	query := `
		SELECT fingerprint, COUNT(DISTINCT user) 
		FROM errors 
		WHERE project_id = ? AND fingerprint IN (` + placeholders + `) 
		AND user IS NOT NULL AND user != '' AND user != '{}' 
		GROUP BY fingerprint`

	args := []interface{}{projectID}
	for _, fp := range fingerprints {
		args = append(args, fp)
	}

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	userCounts := make(map[string]int)
	for rows.Next() {
		var fingerprint string
		var count int
		if err := rows.Scan(&fingerprint, &count); err == nil {
			userCounts[fingerprint] = count
		}
	}

	return userCounts, nil
}

func getTimelinesByFingerprint(db *sql.DB, projectID string, fingerprints []string, duration time.Duration) (map[string][]TimelinePoint, error) {
	if len(fingerprints) == 0 {
		return make(map[string][]TimelinePoint), nil
	}

	placeholders := strings.Repeat("?,", len(fingerprints))
	placeholders = placeholders[:len(placeholders)-1]

	// Get hourly counts for the last 24 hours
	query := `
		SELECT 
			fingerprint,
			strftime('%Y-%m-%d %H:00:00', created_at) as hour,
			COUNT(*) as count
		FROM errors
		WHERE project_id = ? 
		AND fingerprint IN (` + placeholders + `)
		AND created_at >= datetime('now', '-' || ? || ' hours')
		GROUP BY fingerprint, hour
		ORDER BY fingerprint, hour DESC`

	args := []interface{}{projectID}
	for _, fp := range fingerprints {
		args = append(args, fp)
	}
	args = append(args, int(duration.Hours()))

	rows, err := db.Query(query, args...)
	if err != nil {
		return nil, err
	}
	defer rows.Close()

	timelines := make(map[string][]TimelinePoint)
	for rows.Next() {
		var fingerprint, hourStr string
		var count int
		if err := rows.Scan(&fingerprint, &hourStr, &count); err == nil {
			timestamp, _ := time.Parse("2006-01-02 15:04:05", hourStr)
			point := TimelinePoint{
				Timestamp: timestamp,
				Count:     count,
			}
			timelines[fingerprint] = append(timelines[fingerprint], point)
		}
	}

	return timelines, nil
}

// GetErrorGroupByFingerprint returns all occurrences of a specific error group
func GetErrorGroupByFingerprint(db *sql.DB, projectID, fingerprint string, limit int) (map[string]interface{}, []ErrorEvent, error) {
	// Get group summary
	var g ErrorGroup
	err := db.QueryRow(`
		SELECT 
			fingerprint,
			message,
			level,
			status,
			project_id,
			MIN(created_at) as first_seen,
			MAX(created_at) as last_seen,
			COUNT(*) as event_count,
			environment,
			platform,
			MAX(id) as representative_id
		FROM errors
		WHERE project_id = ? AND fingerprint = ?
		GROUP BY fingerprint, message, level, status, project_id, environment, platform`,
		projectID, fingerprint,
	).Scan(
		&g.Fingerprint, &g.Message, &g.Level, &g.Status, &g.ProjectID,
		&g.FirstSeen, &g.LastSeen, &g.EventCount,
		&g.Environment, &g.Platform, &g.RepresentativeID,
	)
	if err != nil {
		return nil, nil, err
	}

	// Get user count
	var userCount int
	db.QueryRow(`
		SELECT COUNT(DISTINCT user) 
		FROM errors 
		WHERE project_id = ? AND fingerprint = ?
		AND user IS NOT NULL AND user != '' AND user != '{}'`,
		projectID, fingerprint,
	).Scan(&userCount)

	// Get all occurrences
	occurrences, err := db.Query(`
		SELECT id, project_id, message, level, environment, release, platform, 
		       timestamp, stacktrace, context, user, tags, status, trace_id, fingerprint, created_at
		FROM errors 
		WHERE project_id = ? AND fingerprint = ?
		ORDER BY created_at DESC
		LIMIT ?`,
		projectID, fingerprint, limit,
	)
	if err != nil {
		return nil, nil, err
	}
	defer occurrences.Close()

	var events []ErrorEvent
	for occurrences.Next() {
		var e ErrorEvent
		err := occurrences.Scan(
			&e.ID, &e.ProjectID, &e.Message, &e.Level, &e.Environment,
			&e.Release, &e.Platform, &e.Timestamp, &e.Stacktrace, &e.Context,
			&e.User, &e.Tags, &e.Status, &e.TraceID, &e.Fingerprint, &e.CreatedAt,
		)
		if err != nil {
			return nil, nil, err
		}
		events = append(events, e)
	}

	groupData := map[string]interface{}{
		"fingerprint":       g.Fingerprint,
		"message":           g.Message,
		"level":             g.Level,
		"status":            g.Status,
		"project_id":        g.ProjectID,
		"first_seen":        g.FirstSeen,
		"last_seen":         g.LastSeen,
		"event_count":       g.EventCount,
		"user_count":        userCount,
		"environment":       g.Environment,
		"platform":          g.Platform,
		"representative_id": g.RepresentativeID,
	}

	return groupData, events, nil
}

// GenerateFingerprintFromStacktrace creates a fingerprint from stacktrace for better grouping
func GenerateFingerprintFromStacktrace(message, level, platform, stacktraceJSON string) string {
	// Parse stacktrace to extract key frames
	var stacktrace map[string]interface{}
	json.Unmarshal([]byte(stacktraceJSON), &stacktrace)

	// Build fingerprint from message, level, and top stack frames
	fingerprintParts := []string{message, level, platform}

	// Extract frames if available
	if frames, ok := stacktrace["frames"].([]interface{}); ok && len(frames) > 0 {
		// Use top 3 frames for fingerprinting
		frameCount := 3
		if len(frames) < frameCount {
			frameCount = len(frames)
		}

		for i := len(frames) - 1; i >= len(frames)-frameCount && i >= 0; i-- {
			if frame, ok := frames[i].(map[string]interface{}); ok {
				if filename, ok := frame["filename"].(string); ok {
					fingerprintParts = append(fingerprintParts, filename)
				}
				if function, ok := frame["function"].(string); ok {
					fingerprintParts = append(fingerprintParts, function)
				}
				if lineno, ok := frame["lineno"].(float64); ok {
					fingerprintParts = append(fingerprintParts, fmt.Sprintf("%d", int(lineno)))
				}
			}
		}
	}

	// Create SHA256 hash
	h := sha256.New()
	h.Write([]byte(strings.Join(fingerprintParts, "|")))
	return fmt.Sprintf("%x", h.Sum(nil))[:16] // Use first 16 chars
}
