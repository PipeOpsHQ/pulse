package main

import (
	"database/sql"
	"fmt"
	"time"
)

// GetErrorGroupsFallback returns grouped errors using message-based grouping
// This is used when the fingerprint column doesn't exist yet (pre-migration)
func GetErrorGroupsFallback(db *sql.DB, projectID string, limit int, cursor string, status string) ([]map[string]interface{}, string, bool, error) {
	// Build the base query for grouped errors using message as fallback
	baseQuery := `
		SELECT
			message || ':' || level as fingerprint,
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

	baseQuery += " GROUP BY message, level, status, project_id, environment, platform"

	// Cursor-based pagination on last_seen using HAVING clause
	if cursor != "" {
		baseQuery += " HAVING MAX(created_at) < ?"
		args = append(args, cursor)
	}

	baseQuery += " ORDER BY last_seen DESC LIMIT ?"
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

	// Get user counts for each group (message-based)
	fingerprints := make([]string, len(groups))
	for i, g := range groups {
		fingerprints[i] = g.Fingerprint
	}

	userCounts, err := getUserCountsByMessageFallback(db, projectID, groups)
	if err != nil {
		fmt.Printf("Warning: failed to get user counts: %v\n", err)
		userCounts = make(map[string]int)
	}

	// Get timeline data for each group
	timelines, err := getTimelinesByMessageFallback(db, projectID, groups, 24*time.Hour)
	if err != nil {
		fmt.Printf("Warning: failed to get timelines: %v\n", err)
		timelines = make(map[string][]TimelinePoint)
	}

	// Build result with all metadata
	result := make([]map[string]interface{}, 0, len(groups))
	for _, g := range groups {
		groupMap := map[string]interface{}{
			"id":                g.RepresentativeID,
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

func getUserCountsByMessageFallback(db *sql.DB, projectID string, groups []ErrorGroup) (map[string]int, error) {
	if len(groups) == 0 {
		return make(map[string]int), nil
	}

	userCounts := make(map[string]int)

	// Query each group separately since we're matching by message + level
	for _, group := range groups {
		var count int
		err := db.QueryRow(`
			SELECT COUNT(DISTINCT user)
			FROM errors
			WHERE project_id = ? AND message = ? AND level = ?
			AND user IS NOT NULL AND user != '' AND user != '{}'`,
			projectID, group.Message, group.Level,
		).Scan(&count)

		if err == nil {
			userCounts[group.Fingerprint] = count
		}
	}

	return userCounts, nil
}

func getTimelinesByMessageFallback(db *sql.DB, projectID string, groups []ErrorGroup, duration time.Duration) (map[string][]TimelinePoint, error) {
	if len(groups) == 0 {
		return make(map[string][]TimelinePoint), nil
	}

	timelines := make(map[string][]TimelinePoint)

	// Query each group separately since we're matching by message + level
	for _, group := range groups {
		query := `
			SELECT
				strftime('%Y-%m-%d %H:00:00', created_at) as hour,
				COUNT(*) as count
			FROM errors
			WHERE project_id = ? AND message = ? AND level = ?
			AND created_at >= datetime('now', '-' || ? || ' hours')
			GROUP BY hour
			ORDER BY hour DESC`

		rows, err := db.Query(query, projectID, group.Message, group.Level, int(duration.Hours()))
		if err != nil {
			continue
		}

		var points []TimelinePoint
		for rows.Next() {
			var hourStr string
			var count int
			if err := rows.Scan(&hourStr, &count); err == nil {
				timestamp, _ := time.Parse("2006-01-02 15:04:05", hourStr)
				points = append(points, TimelinePoint{
					Timestamp: timestamp,
					Count:     count,
				})
			}
		}
		rows.Close()

		timelines[group.Fingerprint] = points
	}

	return timelines, nil
}
