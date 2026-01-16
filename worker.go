package main

import (
	"database/sql"
	"log"
	"net"
	"net/http"
	"strings"
	"time"

	"github.com/google/uuid"
)

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
