package main

import (
	"database/sql"
	"log"
	"net"
	"net/http"
	"strings"
	"sync"
	"time"

	"github.com/google/uuid"
)

// Batch inserter for errors
type ErrorBatch struct {
	Event   *ErrorEvent
	Project *Project
}

var (
	errorBatchChan  = make(chan ErrorBatch, 1000)
	batchMutex      sync.Mutex
	pendingErrors   []ErrorBatch
	lastBatchTime   time.Time
	batchSize       = 100
	batchInterval   = 100 * time.Millisecond
)

func StartErrorBatchInserter(db *sql.DB) {
	log.Println("Starting error batch inserter...")
	ticker := time.NewTicker(batchInterval)
	defer ticker.Stop()

	for {
		select {
		case errorBatch := <-errorBatchChan:
			batchMutex.Lock()
			pendingErrors = append(pendingErrors, errorBatch)
			shouldFlush := len(pendingErrors) >= batchSize
			batchMutex.Unlock()

			if shouldFlush {
				flushErrorBatch(db)
			}
		case <-ticker.C:
			batchMutex.Lock()
			shouldFlush := len(pendingErrors) > 0 && time.Since(lastBatchTime) > batchInterval
			batchMutex.Unlock()

			if shouldFlush {
				flushErrorBatch(db)
			}
		}
	}
}

func flushErrorBatch(db *sql.DB) {
	batchMutex.Lock()
	if len(pendingErrors) == 0 {
		batchMutex.Unlock()
		return
	}

	errorsToInsert := make([]ErrorBatch, len(pendingErrors))
	copy(errorsToInsert, pendingErrors)
	pendingErrors = pendingErrors[:0]
	lastBatchTime = time.Now()
	batchMutex.Unlock()

	if len(errorsToInsert) == 0 {
		return
	}

	// Batch insert errors
	tx, err := db.Begin()
	if err != nil {
		log.Printf("Failed to begin batch transaction: %v", err)
		// Fallback to individual inserts
		for _, eb := range errorsToInsert {
			InsertError(db, eb.Event)
		}
		return
	}

	stmt, err := tx.Prepare(`INSERT INTO errors (id, project_id, message, level, environment, release, platform, timestamp, stacktrace, context, user, tags, status, trace_id, created_at) VALUES (?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?, ?)`)
	if err != nil {
		tx.Rollback()
		log.Printf("Failed to prepare batch statement: %v", err)
		// Fallback to individual inserts
		for _, eb := range errorsToInsert {
			InsertError(db, eb.Event)
		}
		return
	}
	defer stmt.Close()

	projectCounts := make(map[string]int)
	for _, eb := range errorsToInsert {
		_, err := stmt.Exec(
			eb.Event.ID, eb.Event.ProjectID, eb.Event.Message, eb.Event.Level, eb.Event.Environment,
			eb.Event.Release, eb.Event.Platform, eb.Event.Timestamp, eb.Event.Stacktrace, eb.Event.Context,
			eb.Event.User, eb.Event.Tags, eb.Event.Status, eb.Event.TraceID, eb.Event.CreatedAt,
		)
		if err != nil {
			log.Printf("Failed to insert error in batch: %v", err)
			continue
		}
		projectCounts[eb.Event.ProjectID]++

		// Trigger notifications asynchronously
		if eb.Project != nil {
			go triggerNotifications(db, eb.Project, eb.Event)
		}
	}

	// Batch update project counters
	for projectID, count := range projectCounts {
		_, err := tx.Exec("UPDATE projects SET current_month_events = current_month_events + ? WHERE id = ?", count, projectID)
		if err != nil {
			log.Printf("Failed to update project counter for %s: %v", projectID, err)
		}
	}

	if err := tx.Commit(); err != nil {
		log.Printf("Failed to commit batch transaction: %v", err)
		// Fallback to individual inserts
		for _, eb := range errorsToInsert {
			InsertError(db, eb.Event)
		}
		return
	}

	log.Printf("Batch inserted %d errors for %d projects", len(errorsToInsert), len(projectCounts))
}

func StartMonitorWorker(db *sql.DB) {
	log.Println("Starting uptime monitor worker...")
	// Check every 10 seconds for monitors that need checking
	ticker := time.NewTicker(10 * time.Second)
	defer ticker.Stop()

	for range ticker.C {
		checkMonitors(db)
	}
}

func checkMonitors(db *sql.DB) {
	monitors, err := GetAllActiveMonitors(db)
	if err != nil {
		log.Printf("Error fetching monitors: %v", err)
		return
	}

	for _, m := range monitors {
		// Run checks concurrently
		go processMonitor(db, m)
	}
}

func processMonitor(db *sql.DB, m Monitor) {
	// Check if due: if LastCheckedAt exists and time since check < interval, skip
	if m.LastCheckedAt != nil && time.Since(*m.LastCheckedAt).Seconds() < float64(m.Interval) {
		return
	}

	start := time.Now()
	var status string
	var statusCode int
	var errMsg string

	timeout := time.Duration(m.Timeout) * time.Second
	if timeout == 0 {
		timeout = 30 * time.Second // Default timeout
	}

	// Route to appropriate check based on monitor type
	switch strings.ToLower(m.Type) {
	case "http", "https":
		status, statusCode, errMsg = checkHTTP(m.URL, timeout)
	case "tcp":
		status, statusCode, errMsg = checkTCP(m.URL, timeout)
	case "icmp":
		status, statusCode, errMsg = checkICMP(m.URL, timeout)
	case "dns":
		status, statusCode, errMsg = checkDNS(m.URL, timeout)
	default:
		// Default to HTTP for backward compatibility
		status, statusCode, errMsg = checkHTTP(m.URL, timeout)
	}

	duration := time.Since(start).Milliseconds()

	check := &MonitorCheck{
		ID:           uuid.New().String(),
		MonitorID:    m.ID,
		Status:       status,
		ResponseTime: int(duration),
		StatusCode:   statusCode,
		ErrorMessage: errMsg,
		CreatedAt:    time.Now(),
	}

	if err := InsertMonitorCheck(db, check); err != nil {
		log.Printf("Failed to insert check for monitor %s: %v", m.ID, err)
		return
	}

	// Update monitor status
	_, err := db.Exec("UPDATE monitors SET status = ?, last_checked_at = ? WHERE id = ?", status, time.Now(), m.ID)
	if err != nil {
		log.Printf("Failed to update monitor status for %s: %v", m.ID, err)
	}
}

func checkHTTP(url string, timeout time.Duration) (status string, statusCode int, errMsg string) {
	client := http.Client{
		Timeout: timeout,
	}

	resp, err := client.Get(url)
	if err != nil {
		return "down", 0, err.Error()
	}
	defer resp.Body.Close()

	statusCode = resp.StatusCode
	if statusCode >= 200 && statusCode < 300 {
		return "up", statusCode, ""
	}
	return "down", statusCode, resp.Status
}

func checkTCP(target string, timeout time.Duration) (status string, statusCode int, errMsg string) {
	conn, err := net.DialTimeout("tcp", target, timeout)
	if err != nil {
		return "down", 0, err.Error()
	}
	defer conn.Close()
	return "up", 0, ""
}

func checkICMP(hostname string, timeout time.Duration) (status string, statusCode int, errMsg string) {
	// ICMP requires root privileges on most systems, so we'll do a TCP connection check instead
	// This is a simplified check - for true ICMP, you'd need to use raw sockets
	conn, err := net.DialTimeout("tcp", hostname+":80", timeout)
	if err != nil {
		// Try common ports
		ports := []string{"443", "22", "8080"}
		for _, port := range ports {
			conn, err = net.DialTimeout("tcp", hostname+":"+port, timeout)
			if err == nil {
				conn.Close()
				return "up", 0, ""
			}
		}
		return "down", 0, "Host unreachable"
	}
	defer conn.Close()
	return "up", 0, ""
}

func checkDNS(hostname string, timeout time.Duration) (status string, statusCode int, errMsg string) {
	// Check DNS resolution
	_, err := net.LookupHost(hostname)
	if err != nil {
		return "down", 0, err.Error()
	}
	return "up", 0, ""
}
