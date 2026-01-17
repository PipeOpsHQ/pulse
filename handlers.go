package main

import (
	"database/sql"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"strconv"
	"strings"
	"time"

	"bytes"
	"log"

	"github.com/golang-jwt/jwt/v5"
	"github.com/google/uuid"
	"github.com/gorilla/mux"
	"github.com/pquerna/otp/totp"
)

// FlexTimestamp handles timestamps that can be either float64 (Unix) or string (ISO8601)
type FlexTimestamp float64

func (ft *FlexTimestamp) UnmarshalJSON(data []byte) error {
	// Try float64 first
	var f float64
	if err := json.Unmarshal(data, &f); err == nil {
		*ft = FlexTimestamp(f)
		return nil
	}

	// Try string next
	var s string
	if err := json.Unmarshal(data, &s); err == nil {
		// Try RFC3339Nano (best for precision)
		t, err := time.Parse(time.RFC3339Nano, s)
		if err == nil {
			*ft = FlexTimestamp(float64(t.UnixNano()) / 1e9)
			return nil
		}

		// Try RFC3339 (standard)
		t, err = time.Parse(time.RFC3339, s)
		if err == nil {
			*ft = FlexTimestamp(float64(t.UnixNano()) / 1e9)
			return nil
		}

		// Try standard Sentry ISO8601 variations
		formats := []string{
			"2006-01-02T15:04:05.999999999",
			"2006-01-02T15:04:05.999999",
			"2006-01-02T15:04:05.999",
			"2006-01-02T15:04:05",
			"2006-01-02 15:04:05",
		}
		for _, format := range formats {
			t, err = time.Parse(format, s)
			if err == nil {
				*ft = FlexTimestamp(float64(t.UnixNano()) / 1e9)
				return nil
			}
		}
	}

	return fmt.Errorf("failed to unmarshal FlexTimestamp: %s", string(data))
}

type StoreRequest struct {
	Message     string                 `json:"message"`
	Level       string                 `json:"level"`
	Environment string                 `json:"environment"`
	Release     string                 `json:"release"`
	Platform    string                 `json:"platform"`
	Timestamp   *time.Time             `json:"timestamp"`
	Stacktrace  map[string]interface{} `json:"stacktrace"`
	Context     map[string]interface{} `json:"context"`
	User        map[string]interface{} `json:"user"`
	Tags        map[string]interface{} `json:"tags"`
}

// Structs for Envelope Parsing
type EnvelopeHeader struct {
	EventID string `json:"event_id"`
	SentAt  string `json:"sent_at"`
	DSN     string `json:"dsn"`
}

type ItemHeader struct {
	Type        string `json:"type"`
	Length      int    `json:"length"`
	ContentType string `json:"content_type"`
}

type SentryTransaction struct {
	EventID     string `json:"event_id"`
	Type        string `json:"type"`
	Transaction string `json:"transaction"`
	Contexts    struct {
		Trace struct {
			TraceID      string `json:"trace_id"`
			SpanID       string `json:"span_id"`
			ParentSpanID string `json:"parent_span_id"`
			Op           string `json:"op"`
			Status       string `json:"status"`
		} `json:"trace"`
	} `json:"contexts"`
	Spans          []SentrySpan  `json:"spans"`
	StartTimestamp FlexTimestamp `json:"start_timestamp"`
	Timestamp      FlexTimestamp `json:"timestamp"`

	// Exception data (transactions can carry errors)
	Exception   SentryExceptionWrapper `json:"exception"`
	Level       string                 `json:"level"`
	Message     interface{}            `json:"message"`
	Environment string                 `json:"environment"`
	Release     string                 `json:"release"`
	Platform    string                 `json:"platform"`
	User        map[string]interface{} `json:"user"`
	Tags        map[string]interface{} `json:"tags"`
	Extra       map[string]interface{} `json:"extra"`
	SDK         map[string]interface{} `json:"sdk"`
}

type SentrySpan struct {
	SpanID         string                 `json:"span_id"`
	TraceID        string                 `json:"trace_id"`
	ParentSpanID   string                 `json:"parent_span_id"`
	Op             string                 `json:"op"`
	Description    string                 `json:"description"`
	StartTimestamp FlexTimestamp          `json:"start_timestamp"`
	Timestamp      FlexTimestamp          `json:"timestamp"`
	Status         string                 `json:"status"`
	Data           map[string]interface{} `json:"data"`
}

// SentryExceptionWrapper handles both Sentry formats:
// 1. Direct array: "exception": [...]
// 2. Object with values: "exception": {"values": [...]}
type SentryExceptionWrapper struct {
	Values []SentryException `json:"values"`
}

// UnmarshalJSON custom unmarshaler to handle both formats
func (e *SentryExceptionWrapper) UnmarshalJSON(data []byte) error {
	// First try to unmarshal as object with "values" field
	var obj struct {
		Values []SentryException `json:"values"`
	}
	if err := json.Unmarshal(data, &obj); err == nil && len(obj.Values) > 0 {
		e.Values = obj.Values
		return nil
	}

	// If that fails, try to unmarshal as direct array
	var arr []SentryException
	if err := json.Unmarshal(data, &arr); err == nil {
		e.Values = arr
		return nil
	}

	// If both fail, return the array error (more descriptive)
	return json.Unmarshal(data, &arr)
}

// SentryEvent represents the full Sentry event format
type SentryEvent struct {
	EventID        string                 `json:"event_id"`
	Type           string                 `json:"type"`    // "error" or "transaction"
	Message        interface{}            `json:"message"` // Can be string or object
	Level          string                 `json:"level"`
	Exception      SentryExceptionWrapper `json:"exception"`
	Stacktrace     map[string]interface{} `json:"stacktrace"`
	SDK            map[string]interface{} `json:"sdk"`
	Environment    string                 `json:"environment"`
	Release        string                 `json:"release"`
	Platform       string                 `json:"platform"`
	Timestamp      *FlexTimestamp         `json:"timestamp"`
	User           map[string]interface{} `json:"user"`
	Tags           map[string]interface{} `json:"tags"`
	Contexts       map[string]interface{} `json:"contexts"`
	Extra          map[string]interface{} `json:"extra"`
	Transaction    string                 `json:"transaction"`     // For transaction events
	Spans          []SentrySpan           `json:"spans"`           // For transaction events
	StartTimestamp *FlexTimestamp         `json:"start_timestamp"` // For transaction events
}

type SentryException struct {
	Type       string                 `json:"type"`
	Value      string                 `json:"value"`
	Mechanism  map[string]interface{} `json:"mechanism"`
	Stacktrace map[string]interface{} `json:"stacktrace"`
}

func storeError(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Extract API key from X-Sentry-Auth header or query parameter
	authHeader := r.Header.Get("X-Sentry-Auth")
	var apiKey string

	if authHeader != "" {
		// Parse Sentry auth format: Sentry sentry_key=xxx, sentry_version=7
		parts := strings.Split(authHeader, ",")
		for _, part := range parts {
			if strings.Contains(part, "sentry_key=") {
				apiKey = strings.TrimSpace(strings.Split(part, "sentry_key=")[1])
				break
			}
		}
	} else {
		apiKey = r.URL.Query().Get("sentry_key")
	}

	if apiKey == "" {
		http.Error(w, "Missing API key", http.StatusUnauthorized)
		return
	}

	// Get project (for notifications and count)
	project, err := GetProjectByAPIKey(db, apiKey)
	if err != nil {
		http.Error(w, "Invalid API key", http.StatusUnauthorized)
		return
	}

	var req StoreRequest
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Convert stacktrace, context, user, tags to JSON strings
	stacktraceJSON, _ := json.Marshal(req.Stacktrace)
	contextJSON, _ := json.Marshal(req.Context)
	userJSON, _ := json.Marshal(req.User)
	tagsJSON, _ := json.Marshal(req.Tags)

	timestamp := time.Now()
	if req.Timestamp != nil {
		timestamp = *req.Timestamp
	}

	event := &ErrorEvent{
		ID:          uuid.New().String(),
		ProjectID:   project.ID,
		Message:     req.Message,
		Level:       req.Level,
		Environment: req.Environment,
		Release:     req.Release,
		Platform:    req.Platform,
		Timestamp:   timestamp,
		Stacktrace:  string(stacktraceJSON),
		Context:     string(contextJSON),
		User:        string(userJSON),
		Tags:        string(tagsJSON),
		CreatedAt:   time.Now(),
	}

	if err := InsertError(db, event); err != nil {
		http.Error(w, "Failed to store error", http.StatusInternalServerError)
		return
	}

	// Increment count
	IncrementProjectEventCount(db, project.ID)

	// Trigger notifications
	triggerNotifications(db, project, event)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ErrorResponse{ID: event.ID})
}

func getProjects(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	projects, err := GetAllProjects(db)
	if err != nil {
		http.Error(w, "Failed to fetch projects", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(projects)
}

func createProject(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req struct {
		Name string `json:"name"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" {
		http.Error(w, "Project name is required", http.StatusBadRequest)
		return
	}

	project, err := CreateProject(db, req.Name)
	if err != nil {
		http.Error(w, "Failed to create project", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusCreated)
	json.NewEncoder(w).Encode(project)
}

func getProject(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	id := vars["id"]

	project, err := GetProject(db, id)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

func getErrors(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Parse pagination parameters
	limit := 50
	offset := 0
	status := r.URL.Query().Get("status")

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	projectID := r.URL.Query().Get("projectId")
	var errors []map[string]interface{}
	var total int
	var err error

	if projectID != "" {
		errors, total, err = GetErrorsWithStats(db, projectID, limit, offset, status)
	} else {
		errors, total, err = GetAllErrorsWithStats(db, limit, offset, status)
	}

	if err != nil {
		http.Error(w, "Failed to fetch errors", http.StatusInternalServerError)
		return
	}

	// Return paginated response
	response := map[string]interface{}{
		"data": errors,
		"meta": map[string]interface{}{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getProjectErrors(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	projectID := vars["projectId"]

	// Parse pagination parameters
	limit := 50
	offset := 0
	status := r.URL.Query().Get("status")

	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil && parsed > 0 && parsed <= 100 {
			limit = parsed
		}
	}

	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil && parsed >= 0 {
			offset = parsed
		}
	}

	errors, total, err := GetErrors(db, projectID, limit, offset, status)
	if err != nil {
		http.Error(w, "Failed to fetch errors", http.StatusInternalServerError)
		return
	}

	// Return paginated response
	response := map[string]interface{}{
		"data": errors,
		"meta": map[string]interface{}{
			"total":  total,
			"limit":  limit,
			"offset": offset,
		},
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getError(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	id := vars["id"]

	errorEvent, err := GetErrorWithStats(db, id)
	if err != nil {
		http.Error(w, "Error not found", http.StatusNotFound)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(errorEvent)
}

func getErrorOccurrences(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	id := vars["id"]

	// First get the error to get the message and project_id for grouping
	errorEvent, err := GetError(db, id)
	if err != nil {
		http.Error(w, "Error not found", http.StatusNotFound)
		return
	}

	occurrences, err := GetErrorOccurrences(db, errorEvent.Message, errorEvent.ProjectID, 50)
	if err != nil {
		http.Error(w, "Failed to fetch occurrences", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(occurrences)
}

// storeErrorSentry handles Sentry-compatible event ingestion at /api/{project_id}/store/
func storeErrorSentry(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	projectID := vars["projectId"]

	// Extract API key from X-Sentry-Auth header
	authHeader := r.Header.Get("X-Sentry-Auth")
	var apiKey string

	if authHeader != "" {
		// Parse Sentry auth format: Sentry sentry_key=xxx, sentry_version=7
		parts := strings.Split(authHeader, ",")
		for _, part := range parts {
			if strings.Contains(part, "sentry_key=") {
				apiKey = strings.TrimSpace(strings.Split(part, "sentry_key=")[1])
				break
			}
		}
	}

	// Also try X-Pulse-Auth header (simpler format)
	if apiKey == "" {
		apiKey = r.Header.Get("X-Pulse-Auth")
	}

	// Try Authorization header (Bearer token or Basic auth)
	if apiKey == "" {
		auth := r.Header.Get("Authorization")
		if strings.HasPrefix(auth, "Bearer ") {
			apiKey = strings.TrimPrefix(auth, "Bearer ")
		} else if strings.HasPrefix(auth, "Basic ") {
			// Basic auth: base64(key:secret) - for now just extract key
			decoded, err := base64.StdEncoding.DecodeString(strings.TrimPrefix(auth, "Basic "))
			if err == nil {
				parts := strings.Split(string(decoded), ":")
				if len(parts) > 0 {
					apiKey = parts[0]
				}
			}
		}
	}

	// Try query parameter as fallback
	if apiKey == "" {
		apiKey = r.URL.Query().Get("sentry_key")
	}

	if apiKey == "" {
		log.Printf("[DSN Debug] Missing API key for project %s. Method: %s, Path: %s, Headers: %v, URL: %s",
			projectID, r.Method, r.URL.Path, r.Header, r.URL.String())
		http.Error(w, "Missing API key. Please include X-Sentry-Auth header with sentry_key parameter, or use X-Pulse-Auth header", http.StatusUnauthorized)
		return
	}

	// Validate project ID and API key match
	if err := ValidateProjectAndKey(db, projectID, apiKey); err != nil {
		log.Printf("[DSN Debug] Validation failed for project %s: %v", projectID, err)
		http.Error(w, fmt.Sprintf("Invalid project ID or API key: %v", err), http.StatusUnauthorized)
		return
	}

	log.Printf("[DSN Debug] Successfully authenticated request for project %s", projectID)

	// Security Policy Check
	policy, err := GetSecurityPolicy(db, projectID)
	if err == nil && policy.Enforced {
		clientIP := r.Header.Get("X-Forwarded-For")
		if clientIP == "" {
			clientIP = r.RemoteAddr
		}
		// Basic check if IP is in whitelist (if whitelist is set)
		if policy.IPWhitelist != "" && !strings.Contains(policy.IPWhitelist, clientIP) {
			http.Error(w, "Security policy violation: IP not allowed", http.StatusForbidden)
			return
		}
	}

	// Read body first so we can check the type
	bodyBytes, err := io.ReadAll(r.Body)
	if err != nil {
		log.Printf("[DSN Debug] Failed to read body for project %s: %v", projectID, err)
		http.Error(w, "Failed to read request body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Check if this is a transaction event by peeking at the JSON
	var eventTypeCheck struct {
		Type string `json:"type"`
	}
	if err := json.Unmarshal(bodyBytes, &eventTypeCheck); err == nil && eventTypeCheck.Type == "transaction" {
		// Parse as transaction
		var tx SentryTransaction
		if err := json.Unmarshal(bodyBytes, &tx); err != nil {
			log.Printf("[DSN Debug] Failed to parse transaction for project %s: %v", projectID, err)
			http.Error(w, fmt.Sprintf("Invalid transaction format: %v", err), http.StatusBadRequest)
			return
		}

		log.Printf("[DSN Debug] Processing transaction %s for project %s", tx.Transaction, projectID)

		// Convert root transaction to Span
		startTime := time.Now()
		endTime := time.Now()
		if tx.StartTimestamp > 0 {
			startTime = floatToTime(float64(tx.StartTimestamp))
		}
		if tx.Timestamp > 0 {
			endTime = floatToTime(float64(tx.Timestamp))
		}

		rootSpan := &TraceSpan{
			ID:             uuid.New().String(),
			ProjectID:      projectID,
			TraceID:        tx.Contexts.Trace.TraceID,
			SpanID:         tx.Contexts.Trace.SpanID,
			ParentSpanID:   tx.Contexts.Trace.ParentSpanID,
			Name:           tx.Transaction,
			Op:             tx.Contexts.Trace.Op,
			Description:    tx.Transaction,
			StartTimestamp: startTime,
			Timestamp:      endTime,
			Status:         tx.Contexts.Trace.Status,
			Data:           "{}",
		}
		if rootSpan.Op == "" {
			rootSpan.Op = "transaction"
		}
		if rootSpan.Name == "" {
			rootSpan.Name = "transaction"
		}
		if rootSpan.TraceID == "" {
			// Generate trace ID if missing
			rootSpan.TraceID = uuid.New().String()
		}
		if rootSpan.SpanID == "" {
			// Generate span ID if missing
			rootSpan.SpanID = uuid.New().String()
		}

		if err := InsertSpan(db, rootSpan); err != nil {
			log.Printf("[DSN Debug] Failed to insert transaction span: %v", err)
			http.Error(w, "Failed to store transaction", http.StatusInternalServerError)
			return
		}

		log.Printf("[DSN Debug] Successfully stored transaction %s (trace: %s) for project %s", rootSpan.SpanID, rootSpan.TraceID, projectID)

		// Process child spans
		for _, s := range tx.Spans {
			dataJSON, _ := json.Marshal(s.Data)
			childStartTime := floatToTime(float64(s.StartTimestamp))
			childEndTime := floatToTime(float64(s.Timestamp))
			childSpan := &TraceSpan{
				ID:             uuid.New().String(),
				ProjectID:      projectID,
				TraceID:        s.TraceID,
				SpanID:         s.SpanID,
				ParentSpanID:   s.ParentSpanID,
				Name:           s.Description,
				Op:             s.Op,
				Description:    s.Description,
				StartTimestamp: childStartTime,
				Timestamp:      childEndTime,
				Status:         s.Status,
				Data:           string(dataJSON),
			}
			// If child traceID is empty, inherit from parent
			if childSpan.TraceID == "" {
				childSpan.TraceID = rootSpan.TraceID
			}
			// If parent span ID is empty, set to root span
			if childSpan.ParentSpanID == "" {
				childSpan.ParentSpanID = rootSpan.SpanID
			}
			InsertSpan(db, childSpan)
		}

		w.Header().Set("Content-Type", "application/json")
		json.NewEncoder(w).Encode(ErrorResponse{ID: tx.EventID})
		return
	}

	// Parse as regular error event
	var sentryEvent SentryEvent
	if err := json.Unmarshal(bodyBytes, &sentryEvent); err != nil {
		log.Printf("[DSN Debug] Failed to parse request body for project %s: %v", projectID, err)
		http.Error(w, fmt.Sprintf("Invalid request body: %v", err), http.StatusBadRequest)
		return
	}

	log.Printf("[DSN Debug] Successfully parsed event for project %s: message='%s', level='%s', type='%s'",
		projectID, sentryEvent.Message, sentryEvent.Level, sentryEvent.Type)

	// Extract message (can be string or object)
	message := ""
	if msgStr, ok := sentryEvent.Message.(string); ok {
		message = msgStr
	} else if msgObj, ok := sentryEvent.Message.(map[string]interface{}); ok {
		if formatted, ok := msgObj["formatted"].(string); ok {
			message = formatted
		} else if msg, ok := msgObj["message"].(string); ok {
			message = msg
		}
	}

	// If no message but we have exceptions, use exception value
	if message == "" && len(sentryEvent.Exception.Values) > 0 {
		message = sentryEvent.Exception.Values[0].Value
		if sentryEvent.Exception.Values[0].Type != "" {
			message = sentryEvent.Exception.Values[0].Type + ": " + message
		}
	}

	// Extract stacktrace from exception or use top-level stacktrace
	var stacktraceData map[string]interface{}
	if len(sentryEvent.Exception.Values) > 0 && sentryEvent.Exception.Values[0].Stacktrace != nil {
		stacktraceData = sentryEvent.Exception.Values[0].Stacktrace
	} else if sentryEvent.Stacktrace != nil {
		stacktraceData = sentryEvent.Stacktrace
	} else {
		stacktraceData = make(map[string]interface{})
	}

	// Combine contexts and extra into context
	contextData := make(map[string]interface{})
	if sentryEvent.Contexts != nil {
		contextData["contexts"] = sentryEvent.Contexts
	}
	if sentryEvent.Extra != nil {
		contextData["extra"] = sentryEvent.Extra
	}
	if sentryEvent.SDK != nil {
		contextData["sdk"] = sentryEvent.SDK
	}

	// Convert to JSON strings
	stacktraceJSON, _ := json.Marshal(stacktraceData)
	contextJSON, _ := json.Marshal(contextData)
	userJSON, _ := json.Marshal(sentryEvent.User)
	tagsJSON, _ := json.Marshal(sentryEvent.Tags)

	// Use event_id from Sentry or generate new one
	eventID := sentryEvent.EventID
	if eventID == "" {
		eventID = uuid.New().String()
	}

	timestamp := time.Now()
	if sentryEvent.Timestamp != nil {
		timestamp = floatToTime(float64(*sentryEvent.Timestamp))
	}

	level := sentryEvent.Level
	if level == "" {
		level = "error"
	}

	// Get project for notifications
	project, _ := GetProject(db, projectID)

	event := &ErrorEvent{
		ID:          eventID,
		ProjectID:   projectID,
		Message:     message,
		Level:       level,
		Environment: sentryEvent.Environment,
		Release:     sentryEvent.Release,
		Platform:    sentryEvent.Platform,
		Timestamp:   timestamp,
		Stacktrace:  string(stacktraceJSON),
		Context:     string(contextJSON),
		User:        string(userJSON),
		Tags:        string(tagsJSON),
		CreatedAt:   time.Now(),
	}

	if err := InsertError(db, event); err != nil {
		log.Printf("[DSN Debug] Failed to insert error for project %s: %v", projectID, err)
		http.Error(w, "Failed to store error", http.StatusInternalServerError)
		return
	}

	log.Printf("[DSN Debug] Successfully stored error %s for project %s", eventID, projectID)

	// Increment count
	IncrementProjectEventCount(db, projectID)

	// Trigger notifications
	triggerNotifications(db, project, event)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(ErrorResponse{ID: eventID})
}

// Tracing Handlers

func getProjectTraces(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	projectID := vars["projectId"]

	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	query := r.URL.Query().Get("query")

	spans, err := GetProjectRootSpans(db, projectID, query, limit, offset)
	if err != nil {
		http.Error(w, "Failed to fetch traces", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(spans)
}

func getTraceDetails(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	traceID := vars["traceId"]

	spans, err := GetTraceSpans(db, traceID)
	if err != nil {
		http.Error(w, "Failed to fetch trace details", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(spans)
}

// handleEnvelopeSentry processes Sentry Envelopes (Transactions, Spans, etc.)
func handleEnvelopeSentry(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	projectID := vars["projectId"]

	// Auth check (support both headers)
	apiKey := r.Header.Get("X-Pulse-Auth")
	if apiKey == "" {
		authHeader := r.Header.Get("X-Sentry-Auth")
		if authHeader != "" {
			parts := strings.Split(authHeader, ",")
			for _, part := range parts {
				if strings.Contains(part, "sentry_key=") {
					apiKey = strings.TrimSpace(strings.Split(part, "sentry_key=")[1])
					break
				}
			}
		}
	}

	if apiKey == "" {
		// Try query param
		apiKey = r.URL.Query().Get("sentry_key")
	}

	if apiKey == "" {
		http.Error(w, "Missing API key", http.StatusUnauthorized)
		return
	}

	if err := ValidateProjectAndKey(db, projectID, apiKey); err != nil {
		http.Error(w, "Invalid project ID or API key", http.StatusUnauthorized)
		return
	}

	// Read body
	body, err := io.ReadAll(r.Body)
	if err != nil {
		http.Error(w, "Failed to read body", http.StatusBadRequest)
		return
	}
	defer r.Body.Close()

	// Parse Envelope (properly using length headers)
	reader := bytes.NewReader(body)

	// Get project for notifications
	project, _ := GetProject(db, projectID)

	// Read Envelope Header (first line)
	headerLine, err := readEnvelopeLine(reader)
	if err != nil {
		http.Error(w, "Failed to read envelope header", http.StatusBadRequest)
		return
	}
	log.Printf("[DSN Debug] Envelope Header: %s", string(headerLine))

	for reader.Len() > 0 {
		// Read Item Header (one line)
		itemHeaderLine, err := readEnvelopeLine(reader)
		if err != nil {
			break
		}
		if len(bytes.TrimSpace(itemHeaderLine)) == 0 {
			continue
		}

		var itemHeader ItemHeader
		if err := json.Unmarshal(itemHeaderLine, &itemHeader); err != nil {
			log.Printf("[DSN Debug] Failed to unmarshal item header: %v. Line: %s", err, string(itemHeaderLine))
			continue
		}

		// Read payload based on length
		payload := make([]byte, itemHeader.Length)
		n, err := reader.Read(payload)
		if err != nil || n < itemHeader.Length {
			log.Printf("[DSN Debug] Failed to read payload of length %d. Read %d bytes. Error: %v", itemHeader.Length, n, err)
			break
		}

		// Skip possible trailing \n after payload
		if reader.Len() > 0 {
			b, _ := reader.ReadByte()
			if b != '\n' {
				reader.UnreadByte()
			}
		}

		if itemHeader.Type == "transaction" {
			var tx SentryTransaction
			if err := json.Unmarshal(payload, &tx); err != nil {
				log.Printf("[DSN Debug] Failed to unmarshal transaction: %v", err)
				continue
			}

			log.Printf("[DSN Debug] Processing transaction: %s (ID: %s)", tx.Transaction, tx.EventID)
			log.Printf("[DSN Debug] Transaction Exceptions: %d", len(tx.Exception.Values))

			// Convert root transaction to Span
			rootSpan := &TraceSpan{
				ID:             uuid.New().String(),
				ProjectID:      projectID,
				TraceID:        tx.Contexts.Trace.TraceID,
				SpanID:         tx.Contexts.Trace.SpanID,
				ParentSpanID:   tx.Contexts.Trace.ParentSpanID,
				Name:           tx.Transaction,
				Op:             tx.Contexts.Trace.Op,
				Description:    tx.Transaction,
				StartTimestamp: floatToTime(float64(tx.StartTimestamp)),
				Timestamp:      floatToTime(float64(tx.Timestamp)),
				Status:         tx.Contexts.Trace.Status,
				Data:           "{}",
			}
			if rootSpan.Op == "" {
				rootSpan.Op = "transaction"
			}

			if err := InsertSpan(db, rootSpan); err != nil {
				log.Printf("[DSN Debug] Failed to insert root span: %v", err)
			}

			// Process child spans
			for _, s := range tx.Spans {
				dataJSON, _ := json.Marshal(s.Data)
				childSpan := &TraceSpan{
					ID:             uuid.New().String(),
					ProjectID:      projectID,
					TraceID:        s.TraceID,
					SpanID:         s.SpanID,
					ParentSpanID:   s.ParentSpanID,
					Name:           s.Description,
					Op:             s.Op,
					Description:    s.Description,
					StartTimestamp: floatToTime(float64(s.StartTimestamp)),
					Timestamp:      floatToTime(float64(s.Timestamp)),
					Status:         s.Status,
					Data:           string(dataJSON),
				}
				if childSpan.TraceID == "" {
					childSpan.TraceID = tx.Contexts.Trace.TraceID
				}
				InsertSpan(db, childSpan)
			}

			// Store nested exception if present
			if len(tx.Exception.Values) > 0 {
				log.Printf("[DSN Debug] Storing %d exceptions found in transaction", len(tx.Exception.Values))
				// Code to convert tx to ErrorEvent (reusing logic later or here)
				// [Extraction logic preserved]
				message := extractSentryMessage(tx.Message, tx.Exception)
				var stacktraceData map[string]interface{}
				if len(tx.Exception.Values) > 0 {
					stacktraceData = tx.Exception.Values[0].Stacktrace
				}
				if stacktraceData == nil {
					stacktraceData = make(map[string]interface{})
				}

				contextData := map[string]interface{}{"trace_id": tx.Contexts.Trace.TraceID, "span_id": tx.Contexts.Trace.SpanID}
				if tx.Extra != nil {
					contextData["extra"] = tx.Extra
				}
				if tx.SDK != nil {
					contextData["sdk"] = tx.SDK
				}
				stJSON, _ := json.Marshal(stacktraceData)
				ctxJSON, _ := json.Marshal(contextData)
				usrJSON, _ := json.Marshal(tx.User)
				tagsJSON, _ := json.Marshal(tx.Tags)

				errorEvent := &ErrorEvent{
					ID: tx.EventID, ProjectID: projectID, Message: message, Level: tx.Level,
					Environment: tx.Environment, Release: tx.Release, Platform: tx.Platform,
					Timestamp: floatToTime(float64(tx.Timestamp)), Stacktrace: string(stJSON),
					Context: string(ctxJSON), User: string(usrJSON), Tags: string(tagsJSON),
					Status: "unresolved", CreatedAt: time.Now(),
				}
				if errorEvent.Level == "" {
					errorEvent.Level = "error"
				}
				if err := InsertError(db, errorEvent); err != nil {
					log.Printf("[DSN Debug] Failed to insert error from transaction: %v", err)
				} else {
					log.Printf("[DSN Debug] Successfully stored error from transaction %s", tx.EventID)
					IncrementProjectEventCount(db, projectID)
					triggerNotifications(db, project, errorEvent)
				}
			}

		} else if itemHeader.Type == "event" {
			var evt SentryEvent
			if err := json.Unmarshal(payload, &evt); err != nil {
				log.Printf("[DSN Debug] Failed to unmarshal event: %v", err)
				continue
			}

			log.Printf("[DSN Debug] Processing error event: %s (ID: %s)", evt.EventID, evt.EventID)
			log.Printf("[DSN Debug] Event Exceptions: %d", len(evt.Exception.Values))

			message := extractSentryMessage(evt.Message, evt.Exception)
			var stacktraceData map[string]interface{}
			if len(evt.Exception.Values) > 0 {
				stacktraceData = evt.Exception.Values[0].Stacktrace
			}
			if stacktraceData == nil {
				stacktraceData = make(map[string]interface{})
			}

			stJSON, _ := json.Marshal(stacktraceData)
			ctxJSON, _ := json.Marshal(evt.Contexts)
			usrJSON, _ := json.Marshal(evt.User)
			tagsJSON, _ := json.Marshal(evt.Tags)

			var ts time.Time
			if evt.Timestamp != nil {
				ts = floatToTime(float64(*evt.Timestamp))
			} else {
				ts = time.Now()
			}

			errorEvent := &ErrorEvent{
				ID: evt.EventID, ProjectID: projectID, Message: message, Level: evt.Level,
				Environment: evt.Environment, Release: evt.Release, Platform: evt.Platform,
				Timestamp: ts, Stacktrace: string(stJSON), Context: string(ctxJSON),
				User: string(usrJSON), Tags: string(tagsJSON),
				Status: "unresolved", CreatedAt: time.Now(),
			}
			if errorEvent.Level == "" {
				errorEvent.Level = "error"
			}
			if err := InsertError(db, errorEvent); err != nil {
				log.Printf("[DSN Debug] Failed to insert error event: %v", err)
			} else {
				log.Printf("[DSN Debug] Successfully stored error event %s", evt.EventID)
				IncrementProjectEventCount(db, projectID)
				triggerNotifications(db, project, errorEvent)
			}
		}
	}

	w.WriteHeader(http.StatusAccepted)
	json.NewEncoder(w).Encode(map[string]string{"id": "accepted"})
}

func floatToTime(ts float64) time.Time {
	sec := int64(ts)
	nsec := int64((ts - float64(sec)) * 1e9)
	return time.Unix(sec, nsec)
}

// readEnvelopeLine reads a single line from a bytes.Reader
func readEnvelopeLine(reader *bytes.Reader) ([]byte, error) {
	var line []byte
	for {
		b, err := reader.ReadByte()
		if err != nil {
			return line, err
		}
		if b == '\n' {
			return line, nil
		}
		line = append(line, b)
	}
}

// extractSentryMessage handles various message and exception formats to get a descriptive string
func extractSentryMessage(msg interface{}, exception SentryExceptionWrapper) string {
	message := ""

	// 1. Try formatted message or direct string
	if msgStr, ok := msg.(string); ok {
		message = msgStr
	} else if msgObj, ok := msg.(map[string]interface{}); ok {
		if formatted, ok := msgObj["formatted"].(string); ok {
			message = formatted
		} else if m, ok := msgObj["message"].(string); ok {
			message = m
		}
	}

	// 2. If no message, try the first exception
	if message == "" && len(exception.Values) > 0 {
		ex := exception.Values[0]
		message = ex.Value
		if ex.Type != "" {
			if message != "" {
				message = ex.Type + ": " + message
			} else {
				message = ex.Type
			}
		}
	}

	if message == "" {
		message = "Unknown error"
	}

	return message
}

// getProjectDiscovery returns project information for SDK discovery
func getProjectDiscovery(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	projectID := vars["projectId"]

	project, err := GetProject(db, projectID)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	// Return minimal project info (Sentry-compatible format)
	response := map[string]interface{}{
		"id":   project.ID,
		"name": project.Name,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

// deleteError deletes an error by ID
func deleteError(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	id := vars["id"]

	err := DeleteError(db, id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Error not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to delete error", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// deleteProject deletes a project and all its errors
func deleteProject(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	id := vars["id"]

	err := DeleteProject(db, id)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Project not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to delete project", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// updateErrorStatus updates the status of an error
func updateErrorStatus(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		Status string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Status == "" {
		http.Error(w, "Status is required", http.StatusBadRequest)
		return
	}

	err := UpdateErrorStatus(db, id, req.Status)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Error not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to update error status", http.StatusInternalServerError)
		}
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func updateProjectQuota(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	id := vars["id"]

	var req struct {
		Quota             int `json:"quota"`
		MaxEventsPerMonth int `json:"max_events_per_month"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		log.Printf("[Quota Debug] Failed to decode request body: %v", err)
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	log.Printf("[Quota Debug] Updating quota for project %s. Raw: %+v", id, req)

	quota := req.MaxEventsPerMonth
	if quota == 0 && req.Quota != 0 {
		quota = req.Quota
	}

	if quota < 0 {
		http.Error(w, "Quota must be non-negative", http.StatusBadRequest)
		return
	}

	res, err := db.Exec("UPDATE projects SET max_events_per_month = ? WHERE id = ?", quota, id)
	if err != nil {
		http.Error(w, "Failed to update project quota", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := res.RowsAffected()
	if rowsAffected == 0 {
		http.Error(w, "Project not found or no change made", http.StatusNotFound)
		return
	}

	// Return the updated project
	project, err := GetProject(db, id)
	if err != nil {
		w.WriteHeader(http.StatusNoContent)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(project)
}

// healthCheck returns a 200 OK status
func healthCheck(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Content-Type", "application/json")
	w.WriteHeader(http.StatusOK)
	json.NewEncoder(w).Encode(map[string]string{"status": "ok", "product": "Pulse"})
}

func getSettings(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	settings, err := GetAllSettings(db)
	if err != nil {
		http.Error(w, "Failed to fetch settings", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func updateSettings(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	var req map[string]string
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	for k, v := range req {
		if err := UpdateSetting(db, k, v); err != nil {
			http.Error(w, "Failed to update setting: "+k, http.StatusInternalServerError)
			return
		}
	}

	w.WriteHeader(http.StatusNoContent)
}

func runCleanup(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	retentionStr, _ := GetSetting(db, "retention_days")
	retentionDays := 30
	if retentionStr != "" {
		if val, err := strconv.Atoi(retentionStr); err == nil {
			retentionDays = val
		}
	}

	cutoff := time.Now().AddDate(0, 0, -retentionDays)
	result, err := db.Exec("DELETE FROM errors WHERE created_at < ?", cutoff)
	if err != nil {
		http.Error(w, "Failed to run cleanup", http.StatusInternalServerError)
		return
	}

	rowsAffected, _ := result.RowsAffected()
	log.Printf("System cleanup: Deleted %d old errors (older than %d days)", rowsAffected, retentionDays)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"deleted": rowsAffected,
		"days":    retentionDays,
	})
}

func uploadProjectCoverage(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	projectID := vars["projectId"]

	// Extract API key from X-Pulse-Auth or X-Sentry-Auth header
	apiKey := r.Header.Get("X-Pulse-Auth")
	if apiKey == "" {
		apiKey = r.Header.Get("X-Sentry-Auth")
	}

	if apiKey == "" {
		http.Error(w, "Missing API key", http.StatusUnauthorized)
		return
	}

	// Validate project ID and API key match
	if err := ValidateProjectAndKey(db, projectID, apiKey); err != nil {
		http.Error(w, "Invalid project ID or API key", http.StatusUnauthorized)
		return
	}

	var coverage float64
	var files []FileCoverage

	// Support both JSON (percentage only) and multipart (full file)
	contentType := r.Header.Get("Content-Type")
	if strings.HasPrefix(contentType, "multipart/form-data") {
		err := r.ParseMultipartForm(10 << 20) // 10MB limit
		if err != nil {
			http.Error(w, "Failed to parse multipart form", http.StatusBadRequest)
			return
		}

		file, header, err := r.FormFile("file")
		if err != nil {
			http.Error(w, "Missing 'file' in multipart form", http.StatusBadRequest)
			return
		}
		defer file.Close()

		// Detect format
		filename := strings.ToLower(header.Filename)
		var parseErr error
		if strings.HasSuffix(filename, "coverage.out") || strings.Contains(filename, "cover") {
			coverage, files, parseErr = ParseGoCoverage(file)
		} else if strings.HasSuffix(filename, "lcov.info") || strings.Contains(filename, "lcov") {
			coverage, files, parseErr = ParseLCOVCoverage(file)
		} else {
			http.Error(w, "Unsupported coverage format. Use .out for Go or .info for LCOV", http.StatusBadRequest)
			return
		}

		if parseErr != nil {
			http.Error(w, "Failed to parse coverage file: "+parseErr.Error(), http.StatusInternalServerError)
			return
		}
	} else {
		var req struct {
			Coverage float64 `json:"coverage"`
		}
		if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
			http.Error(w, "Invalid request body", http.StatusBadRequest)
			return
		}
		coverage = req.Coverage
	}

	if err := UpdateProjectCoverage(db, projectID, coverage, files); err != nil {
		http.Error(w, "Failed to update coverage", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func getProjectFileCoverage(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	snapshotID := vars["snapshotId"]

	files, err := GetProjectFileCoverage(db, snapshotID)
	if err != nil {
		http.Error(w, "Failed to fetch file breakdown: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(files)
}

func getProjectCoverageHistory(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	projectID := vars["projectId"]

	history, err := GetProjectCoverageHistory(db, projectID, 30) // Last 30 points
	if err != nil {
		http.Error(w, "Failed to fetch history: "+err.Error(), http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func getCoverageBadge(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	projectID := vars["projectId"]

	project, err := GetProject(db, projectID)
	if err != nil {
		http.Error(w, "Project not found", http.StatusNotFound)
		return
	}

	coverage := project.Coverage
	color := "#10b981" // emerald-500
	if coverage < 50 {
		color = "#ef4444" // red-500
	} else if coverage < 80 {
		color = "#f59e0b" // amber-500
	}

	// Simple SVG badge
	w.Header().Set("Content-Type", "image/svg+xml")
	w.Header().Set("Cache-Control", "no-cache, no-store, must-revalidate")

	coverageText := fmt.Sprintf("%.1f%%", coverage)
	if coverage == 0 {
		coverageText = "N/A"
	}

	fmt.Fprintf(w, `<svg xmlns="http://www.w3.org/2000/svg" width="104" height="20">
	<linearGradient id="b" x2="0" y2="100%%">
		<stop offset="0" stop-color="#bbb" stop-opacity=".1"/>
		<stop offset="1" stop-opacity=".1"/>
	</linearGradient>
	<mask id="a">
		<rect width="104" height="20" rx="3" fill="#fff"/>
	</mask>
	<g mask="url(#a)">
		<path fill="#555" d="M0 0h67v20H0z"/>
		<path fill="%s" d="M67 0h37v20H67z"/>
		<path fill="url(#b)" d="M0 0h104v20H0z"/>
	</g>
	<g fill="#fff" text-anchor="middle" font-family="DejaVu Sans,Verdana,Geneva,sans-serif" font-size="11">
		<text x="33.5" y="15" fill="#010101" fill-opacity=".3">coverage</text>
		<text x="33.5" y="14">coverage</text>
		<text x="85.5" y="15" fill="#010101" fill-opacity=".3">%s</text>
		<text x="85.5" y="14">%s</text>
	</g>
</svg>`, color, coverageText, coverageText)
}

func triggerNotifications(db *sql.DB, project *Project, event *ErrorEvent) {
	settings, err := GetAllSettings(db)
	if err != nil {
		return
	}

	// Slack Notification
	if webhook, ok := settings["slack_webhook"]; ok && webhook != "" {
		go func() {
			payload := map[string]interface{}{
				"text": "*Pulse Alert:* New error in project *" + project.Name + "*\n> " + event.Message,
			}
			body, _ := json.Marshal(payload)
			resp, err := http.Post(webhook, "application/json", bytes.NewBuffer(body))
			if err != nil {
				log.Printf("Failed to send Slack notification: %v", err)
			} else {
				resp.Body.Close()
			}
		}()
	}

	// Generic Webhook
	if webhook, ok := settings["generic_webhook"]; ok && webhook != "" {
		go func() {
			body, _ := json.Marshal(event)
			resp, err := http.Post(webhook, "application/json", bytes.NewBuffer(body))
			if err != nil {
				log.Printf("Failed to send generic webhook: %v", err)
			} else {
				resp.Body.Close()
			}
		}()
	}
}

// Security Vault Handlers

func rotateProjectAPIKey(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	projectID := vars["id"]

	newKey, err := RotateAPIKey(db, projectID)
	if err != nil {
		if err == sql.ErrNoRows {
			http.Error(w, "Project not found", http.StatusNotFound)
		} else {
			http.Error(w, "Failed to rotate API key", http.StatusInternalServerError)
		}
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{"api_key": newKey})
}

func getProjectAPIKeyHistory(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	projectID := vars["id"]

	history, err := GetAPIKeyHistory(db, projectID)
	if err != nil {
		http.Error(w, "Failed to fetch key history", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(history)
}

func getSecurityPolicies(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	projectID := vars["id"]

	policy, err := GetSecurityPolicy(db, projectID)
	if err != nil {
		http.Error(w, "Failed to fetch security policies", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(policy)
}

func updateSecurityPolicies(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	projectID := vars["id"]

	var policy SecurityPolicy
	if err := json.NewDecoder(r.Body).Decode(&policy); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}
	policy.ProjectID = projectID

	if err := UpdateSecurityPolicy(db, &policy); err != nil {
		http.Error(w, "Failed to update security policies", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func setupMFA(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	// Assume user is authenticated
	authHeader := r.Header.Get("Authorization")
	tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
	claims := &Claims{}
	jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return getJWTSecret(), nil
	})

	user, err := GetUserByID(db, claims.UserID)
	if err != nil {
		http.Error(w, "User not found", http.StatusUnauthorized)
		return
	}

	key, err := totp.Generate(totp.GenerateOpts{
		Issuer:      "Pulse",
		AccountName: user.Email,
	})
	if err != nil {
		http.Error(w, "Failed to generate MFA key", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]string{
		"secret": key.Secret(),
		"url":    key.URL(),
	})
}

func enableMFA(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	authHeader := r.Header.Get("Authorization")
	tokenString := strings.Replace(authHeader, "Bearer ", "", 1)
	claims := &Claims{}
	jwt.ParseWithClaims(tokenString, claims, func(token *jwt.Token) (interface{}, error) {
		return getJWTSecret(), nil
	})

	var req struct {
		Secret string `json:"secret"`
		Code   string `json:"code"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	valid := totp.Validate(req.Code, req.Secret)
	if !valid {
		http.Error(w, "Invalid MFA code", http.StatusUnauthorized)
		return
	}

	if err := UpdateUserMFA(db, claims.UserID, true, req.Secret); err != nil {
		http.Error(w, "Failed to enable MFA", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

// Uptime Monitoring Handlers

func getProjectMonitors(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	projectID := vars["id"]

	monitors, err := GetProjectMonitors(db, projectID)
	if err != nil {
		log.Printf("Error getting monitors: %v", err)
		http.Error(w, "Failed to get monitors", http.StatusInternalServerError)
		return
	}

	// Ensure we return an empty array instead of null
	if monitors == nil {
		monitors = []Monitor{}
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(monitors)
}

func createMonitor(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	projectID := vars["id"]

	var req struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		URL      string `json:"url"`
		Interval int    `json:"interval"`
		Timeout  int    `json:"timeout"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	if req.Name == "" || req.URL == "" {
		http.Error(w, "Name and URL/target are required", http.StatusBadRequest)
		return
	}
	if req.Type == "" {
		req.Type = "http"
	}
	// Validate monitor type
	validTypes := map[string]bool{"http": true, "https": true, "tcp": true, "icmp": true, "dns": true}
	if !validTypes[strings.ToLower(req.Type)] {
		http.Error(w, "Invalid monitor type. Supported: http, https, tcp, icmp, dns", http.StatusBadRequest)
		return
	}
	if req.Interval < 60 {
		req.Interval = 60 // Minimum 1 minute
	}
	if req.Timeout < 5 {
		req.Timeout = 30 // Default timeout 30 seconds
	}
	if req.Timeout > 300 {
		req.Timeout = 300 // Maximum timeout 5 minutes
	}

	monitor := &Monitor{
		ID:        uuid.New().String(),
		ProjectID: projectID,
		Name:      req.Name,
		Type:      strings.ToLower(req.Type),
		URL:       req.URL,
		Interval:  req.Interval,
		Timeout:   req.Timeout,
		Status:    "up", // Default status, will be updated by check
		CreatedAt: time.Now(),
	}

	if err := CreateMonitor(db, monitor); err != nil {
		log.Printf("Error creating monitor: %v", err)
		http.Error(w, "Failed to create monitor", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(monitor)
}

func updateMonitor(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	monitorID := vars["monitorId"]

	var req struct {
		Name     string `json:"name"`
		Type     string `json:"type"`
		URL      string `json:"url"`
		Interval int    `json:"interval"`
		Timeout  int    `json:"timeout"`
		Status   string `json:"status"`
	}
	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Build update query dynamically based on provided fields
	updates := []string{}
	args := []interface{}{}

	if req.Name != "" {
		updates = append(updates, "name = ?")
		args = append(args, req.Name)
	}
	if req.Type != "" {
		validTypes := map[string]bool{"http": true, "https": true, "tcp": true, "icmp": true, "dns": true}
		if !validTypes[strings.ToLower(req.Type)] {
			http.Error(w, "Invalid monitor type. Supported: http, https, tcp, icmp, dns", http.StatusBadRequest)
			return
		}
		updates = append(updates, "type = ?")
		args = append(args, strings.ToLower(req.Type))
	}
	if req.URL != "" {
		updates = append(updates, "url = ?")
		args = append(args, req.URL)
	}
	if req.Interval > 0 {
		if req.Interval < 60 {
			req.Interval = 60
		}
		updates = append(updates, "interval = ?")
		args = append(args, req.Interval)
	}
	if req.Timeout > 0 {
		if req.Timeout < 5 {
			req.Timeout = 30
		}
		if req.Timeout > 300 {
			req.Timeout = 300
		}
		updates = append(updates, "timeout = ?")
		args = append(args, req.Timeout)
	}
	if req.Status != "" {
		updates = append(updates, "status = ?")
		args = append(args, req.Status)
	}

	if len(updates) == 0 {
		http.Error(w, "No fields to update", http.StatusBadRequest)
		return
	}

	args = append(args, monitorID)
	query := "UPDATE monitors SET " + strings.Join(updates, ", ") + " WHERE id = ?"
	_, err := db.Exec(query, args...)
	if err != nil {
		log.Printf("Error updating monitor: %v", err)
		http.Error(w, "Failed to update monitor", http.StatusInternalServerError)
		return
	}

	// Fetch updated monitor
	var m Monitor
	var lastChecked sql.NullTime
	var timeout sql.NullInt64
	err = db.QueryRow("SELECT id, project_id, name, type, url, interval, timeout, status, last_checked_at, created_at FROM monitors WHERE id = ?", monitorID).
		Scan(&m.ID, &m.ProjectID, &m.Name, &m.Type, &m.URL, &m.Interval, &timeout, &m.Status, &lastChecked, &m.CreatedAt)
	if err != nil {
		log.Printf("Error fetching updated monitor: %v", err)
		http.Error(w, "Failed to fetch updated monitor", http.StatusInternalServerError)
		return
	}
	if lastChecked.Valid {
		m.LastCheckedAt = &lastChecked.Time
	}
	if timeout.Valid {
		m.Timeout = int(timeout.Int64)
	} else {
		m.Timeout = 30
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(m)
}

func deleteMonitor(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	monitorID := vars["monitorId"]

	_, err := db.Exec("DELETE FROM monitors WHERE id = ?", monitorID)
	if err != nil {
		log.Printf("Error deleting monitor: %v", err)
		http.Error(w, "Failed to delete monitor", http.StatusInternalServerError)
		return
	}

	w.WriteHeader(http.StatusNoContent)
}

func getStatusPage(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	projectID := vars["projectId"]

	monitors, err := GetProjectMonitors(db, projectID)
	if err != nil {
		http.Error(w, "Failed to fetch monitors", http.StatusInternalServerError)
		return
	}

	// Calculate uptime for each monitor
	type StatusPageMonitor struct {
		Monitor
		Uptime24h    float64        `json:"uptime_24h"`
		Uptime7d     float64        `json:"uptime_7d"`
		Uptime30d    float64        `json:"uptime_30d"`
		RecentChecks []MonitorCheck `json:"recent_checks"`
	}

	statusMonitors := make([]StatusPageMonitor, 0, len(monitors))
	for _, m := range monitors {
		checks, _ := GetMonitorChecks(db, m.ID, 1000)

		now := time.Now()
		dayAgo := now.Add(-24 * time.Hour)
		weekAgo := now.Add(-7 * 24 * time.Hour)
		monthAgo := now.Add(-30 * 24 * time.Hour)

		var up24h, total24h, up7d, total7d, up30d, total30d int

		for _, check := range checks {
			if check.CreatedAt.After(monthAgo) {
				total30d++
				if check.Status == "up" {
					up30d++
				}
				if check.CreatedAt.After(weekAgo) {
					total7d++
					if check.Status == "up" {
						up7d++
					}
					if check.CreatedAt.After(dayAgo) {
						total24h++
						if check.Status == "up" {
							up24h++
						}
					}
				}
			}
		}

		var uptime24h, uptime7d, uptime30d float64
		if total24h > 0 {
			uptime24h = float64(up24h) / float64(total24h) * 100
		}
		if total7d > 0 {
			uptime7d = float64(up7d) / float64(total7d) * 100
		}
		if total30d > 0 {
			uptime30d = float64(up30d) / float64(total30d) * 100
		}

		recentChecks := checks
		if len(checks) > 50 {
			recentChecks = checks[:50]
		}

		statusMonitors = append(statusMonitors, StatusPageMonitor{
			Monitor:      m,
			Uptime24h:    uptime24h,
			Uptime7d:     uptime7d,
			Uptime30d:    uptime30d,
			RecentChecks: recentChecks,
		})
	}

	// Get project name
	project, _ := GetProject(db, projectID)

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(map[string]interface{}{
		"project": map[string]interface{}{
			"id":   projectID,
			"name": project.Name,
		},
		"monitors": statusMonitors,
	})
}

// Insights Handler - Comprehensive data aggregation
func getInsights(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	projectID := r.URL.Query().Get("projectId")
	timeRange := r.URL.Query().Get("range") // 24h, 7d, 30d
	if timeRange == "" {
		timeRange = "7d"
	}

	trends, _ := GetHourlyStats(db, projectID)

	insights := map[string]interface{}{
		"time_range":   timeRange,
		"generated_at": time.Now(),
		"trends":       trends,
	}

	// Error Tracking Stats
	errorStats := map[string]interface{}{}
	if projectID != "" {
		errors, total, _ := GetErrorsWithStats(db, projectID, 1000, 0, "")
		errorStats["total_errors"] = total
		errorStats["recent_errors"] = len(errors)

		// Count by level
		levelCounts := map[string]int{"error": 0, "warning": 0, "info": 0, "fatal": 0}
		statusCounts := map[string]int{"unresolved": 0, "resolved": 0, "ignored": 0}

		for _, err := range errors {
			if level, ok := err["level"].(string); ok {
				if count, exists := levelCounts[level]; exists {
					levelCounts[level] = count + 1
				}
			}
			if status, ok := err["status"].(string); ok {
				if count, exists := statusCounts[status]; exists {
					statusCounts[status] = count + 1
				}
			}
		}
		errorStats["by_level"] = levelCounts
		errorStats["by_status"] = statusCounts

		// Recent errors (last 10)
		recentErrors := []map[string]interface{}{}
		for i, err := range errors {
			if i >= 10 {
				break
			}
			recentErrors = append(recentErrors, err)
		}
		errorStats["recent"] = recentErrors
	} else {
		errors, total, _ := GetAllErrorsWithStats(db, 1000, 0, "")
		errorStats["total_errors"] = total
		errorStats["recent_errors"] = len(errors)

		levelCounts := map[string]int{"error": 0, "warning": 0, "info": 0, "fatal": 0}
		statusCounts := map[string]int{"unresolved": 0, "resolved": 0, "ignored": 0}

		for _, err := range errors {
			if level, ok := err["level"].(string); ok {
				if count, exists := levelCounts[level]; exists {
					levelCounts[level] = count + 1
				}
			}
			if status, ok := err["status"].(string); ok {
				if count, exists := statusCounts[status]; exists {
					statusCounts[status] = count + 1
				}
			}
		}
		errorStats["by_level"] = levelCounts
		errorStats["by_status"] = statusCounts

		recentErrors := []map[string]interface{}{}
		for i, err := range errors {
			if i >= 10 {
				break
			}
			recentErrors = append(recentErrors, err)
		}
		errorStats["recent"] = recentErrors
	}
	insights["errors"] = errorStats

	// Traces/Performance Stats
	traceStats := map[string]interface{}{}
	if projectID != "" {
		spans, _ := GetProjectRootSpans(db, projectID, "", 1000, 0)
		traceStats["total_traces"] = len(spans)

		// Calculate average duration
		var totalDuration int64
		var count int
		for _, span := range spans {
			if !span.StartTimestamp.IsZero() && !span.Timestamp.IsZero() {
				duration := span.Timestamp.Sub(span.StartTimestamp).Milliseconds()
				totalDuration += duration
				count++
			}
		}
		if count > 0 {
			traceStats["avg_duration_ms"] = totalDuration / int64(count)
		} else {
			traceStats["avg_duration_ms"] = 0
		}

		// Recent traces
		recentTraces := []TraceSpan{}
		for i, span := range spans {
			if i >= 10 {
				break
			}
			recentTraces = append(recentTraces, span)
		}
		traceStats["recent"] = recentTraces
	} else {
		// Get all traces across projects
		var allSpans []TraceSpan
		projects, _ := GetAllProjects(db)
		for _, p := range projects {
			spans, _ := GetProjectRootSpans(db, p.ID, "", 100, 0)
			allSpans = append(allSpans, spans...)
		}
		traceStats["total_traces"] = len(allSpans)

		var totalDuration int64
		var count int
		for _, span := range allSpans {
			if !span.StartTimestamp.IsZero() && !span.Timestamp.IsZero() {
				duration := span.Timestamp.Sub(span.StartTimestamp).Milliseconds()
				totalDuration += duration
				count++
			}
		}
		if count > 0 {
			traceStats["avg_duration_ms"] = totalDuration / int64(count)
		} else {
			traceStats["avg_duration_ms"] = 0
		}

		recentTraces := []TraceSpan{}
		for i, span := range allSpans {
			if i >= 10 {
				break
			}
			recentTraces = append(recentTraces, span)
		}
		traceStats["recent"] = recentTraces
	}
	insights["traces"] = traceStats

	// Uptime Stats
	uptimeStats := map[string]interface{}{}
	if projectID != "" {
		monitors, _ := GetProjectMonitors(db, projectID)
		uptimeStats["total_monitors"] = len(monitors)

		var totalUptime24h, totalUptime7d, totalUptime30d float64
		var activeMonitors int

		for _, m := range monitors {
			checks, _ := GetMonitorChecks(db, m.ID, 1000)
			if len(checks) == 0 {
				continue
			}
			activeMonitors++

			now := time.Now()
			dayAgo := now.Add(-24 * time.Hour)
			weekAgo := now.Add(-7 * 24 * time.Hour)
			monthAgo := now.Add(-30 * 24 * time.Hour)

			var up24h, total24h, up7d, total7d, up30d, total30d int
			for _, check := range checks {
				if check.CreatedAt.After(monthAgo) {
					total30d++
					if check.Status == "up" {
						up30d++
					}
					if check.CreatedAt.After(weekAgo) {
						total7d++
						if check.Status == "up" {
							up7d++
						}
						if check.CreatedAt.After(dayAgo) {
							total24h++
							if check.Status == "up" {
								up24h++
							}
						}
					}
				}
			}

			if total24h > 0 {
				totalUptime24h += float64(up24h) / float64(total24h) * 100
			}
			if total7d > 0 {
				totalUptime7d += float64(up7d) / float64(total7d) * 100
			}
			if total30d > 0 {
				totalUptime30d += float64(up30d) / float64(total30d) * 100
			}
		}

		if activeMonitors > 0 {
			uptimeStats["avg_uptime_24h"] = totalUptime24h / float64(activeMonitors)
			uptimeStats["avg_uptime_7d"] = totalUptime7d / float64(activeMonitors)
			uptimeStats["avg_uptime_30d"] = totalUptime30d / float64(activeMonitors)
		} else {
			uptimeStats["avg_uptime_24h"] = 0
			uptimeStats["avg_uptime_7d"] = 0
			uptimeStats["avg_uptime_30d"] = 0
		}

		uptimeStats["monitors"] = monitors
	} else {
		// All projects
		projects, _ := GetAllProjects(db)
		var totalMonitors int
		var totalUptime24h, totalUptime7d, totalUptime30d float64
		var activeMonitors int

		for _, p := range projects {
			monitors, _ := GetProjectMonitors(db, p.ID)
			totalMonitors += len(monitors)

			for _, m := range monitors {
				checks, _ := GetMonitorChecks(db, m.ID, 1000)
				if len(checks) == 0 {
					continue
				}
				activeMonitors++

				now := time.Now()
				dayAgo := now.Add(-24 * time.Hour)
				weekAgo := now.Add(-7 * 24 * time.Hour)
				monthAgo := now.Add(-30 * 24 * time.Hour)

				var up24h, total24h, up7d, total7d, up30d, total30d int
				for _, check := range checks {
					if check.CreatedAt.After(monthAgo) {
						total30d++
						if check.Status == "up" {
							up30d++
						}
						if check.CreatedAt.After(weekAgo) {
							total7d++
							if check.Status == "up" {
								up7d++
							}
							if check.CreatedAt.After(dayAgo) {
								total24h++
								if check.Status == "up" {
									up24h++
								}
							}
						}
					}
				}

				if total24h > 0 {
					totalUptime24h += float64(up24h) / float64(total24h) * 100
				}
				if total7d > 0 {
					totalUptime7d += float64(up7d) / float64(total7d) * 100
				}
				if total30d > 0 {
					totalUptime30d += float64(up30d) / float64(total30d) * 100
				}
			}
		}

		uptimeStats["total_monitors"] = totalMonitors
		if activeMonitors > 0 {
			uptimeStats["avg_uptime_24h"] = totalUptime24h / float64(activeMonitors)
			uptimeStats["avg_uptime_7d"] = totalUptime7d / float64(activeMonitors)
			uptimeStats["avg_uptime_30d"] = totalUptime30d / float64(activeMonitors)
		} else {
			uptimeStats["avg_uptime_24h"] = 0
			uptimeStats["avg_uptime_7d"] = 0
			uptimeStats["avg_uptime_30d"] = 0
		}
	}
	insights["uptime"] = uptimeStats

	// Coverage Stats
	coverageStats := map[string]interface{}{}
	if projectID != "" {
		project, _ := GetProject(db, projectID)
		if project != nil {
			coverageStats["current_coverage"] = project.Coverage
			coverageStats["last_updated"] = project.CoverageUpdatedAt

			history, _ := GetProjectCoverageHistory(db, projectID, 30)
			coverageStats["history_count"] = len(history)

			if len(history) >= 2 {
				delta := history[0].Percentage - history[1].Percentage
				direction := "stable"
				if delta > 0 {
					direction = "up"
				} else if delta < 0 {
					direction = "down"
				}
				coverageStats["trend"] = map[string]interface{}{
					"delta":     delta,
					"direction": direction,
				}
			}
			if len(history) > 10 {
				coverageStats["recent_history"] = history[:10]
			} else {
				coverageStats["recent_history"] = history
			}
		}
	} else {
		projects, _ := GetAllProjects(db)
		var totalCoverage float64
		var projectsWithCoverage int
		var allHistory []CoverageSnapshot

		for _, p := range projects {
			if p.Coverage > 0 {
				totalCoverage += p.Coverage
				projectsWithCoverage++
			}
			history, _ := GetProjectCoverageHistory(db, p.ID, 5)
			allHistory = append(allHistory, history...)
		}

		if projectsWithCoverage > 0 {
			coverageStats["avg_coverage"] = totalCoverage / float64(projectsWithCoverage)
		} else {
			coverageStats["avg_coverage"] = 0
		}
		coverageStats["projects_with_coverage"] = projectsWithCoverage
		coverageStats["total_projects"] = len(projects)
		if len(allHistory) > 20 {
			coverageStats["recent_history"] = allHistory[:20]
		} else {
			coverageStats["recent_history"] = allHistory
		}
	}
	insights["coverage"] = coverageStats

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(insights)
}

func getProjectSettings(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	projectID := vars["id"]

	settings, err := GetProjectSettings(db, projectID)
	if err != nil {
		log.Printf("Error fetching project settings: %v", err)
		http.Error(w, "Failed to fetch project settings", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(settings)
}

func updateProjectSettings(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	vars := mux.Vars(r)
	projectID := vars["id"]

	var req struct {
		Name                   string `json:"name"`
		NotificationEnabled    *bool  `json:"notification_enabled"`
		NotificationLevels     string `json:"notification_levels"`
		NotificationFrequency  string `json:"notification_frequency"`
		NotificationEmail      string `json:"notification_email"`
		NotificationWebhookURL string `json:"notification_webhook_url"`
		NotificationRateLimit  *int   `json:"notification_rate_limit"`
	}

	if err := json.NewDecoder(r.Body).Decode(&req); err != nil {
		http.Error(w, "Invalid request body", http.StatusBadRequest)
		return
	}

	// Update project name if provided
	if req.Name != "" {
		if err := UpdateProjectName(db, projectID, req.Name); err != nil {
			log.Printf("Error updating project name: %v", err)
			http.Error(w, "Failed to update project name", http.StatusInternalServerError)
			return
		}
	}

	// Get current settings
	settings, err := GetProjectSettings(db, projectID)
	if err != nil {
		log.Printf("Error fetching project settings: %v", err)
		http.Error(w, "Failed to fetch project settings", http.StatusInternalServerError)
		return
	}

	// Update settings fields if provided
	if req.NotificationEnabled != nil {
		settings.NotificationEnabled = *req.NotificationEnabled
	}
	if req.NotificationLevels != "" {
		settings.NotificationLevels = req.NotificationLevels
	}
	if req.NotificationFrequency != "" {
		settings.NotificationFrequency = req.NotificationFrequency
	}
	if req.NotificationEmail != "" || req.NotificationEmail == "" {
		settings.NotificationEmail = req.NotificationEmail
	}
	if req.NotificationWebhookURL != "" || req.NotificationWebhookURL == "" {
		settings.NotificationWebhookURL = req.NotificationWebhookURL
	}
	if req.NotificationRateLimit != nil {
		settings.NotificationRateLimit = *req.NotificationRateLimit
	}

	settings.ProjectID = projectID

	if err := UpdateProjectSettings(db, settings); err != nil {
		log.Printf("Error updating project settings: %v", err)
		http.Error(w, "Failed to update project settings", http.StatusInternalServerError)
		return
	}

	// Return updated project and settings
	project, err := GetProject(db, projectID)
	if err != nil {
		log.Printf("Error fetching project: %v", err)
		http.Error(w, "Failed to fetch project", http.StatusInternalServerError)
		return
	}

	response := map[string]interface{}{
		"project":  project,
		"settings": settings,
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(response)
}

func getSystemStats(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	stats, err := GetSystemStats(db)
	if err != nil {
		log.Printf("Error fetching system stats: %v", err)
		http.Error(w, "Failed to fetch system stats", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(stats)
}

func getAllTraces(w http.ResponseWriter, r *http.Request, db *sql.DB) {
	limit := 50
	if l := r.URL.Query().Get("limit"); l != "" {
		if parsed, err := strconv.Atoi(l); err == nil {
			limit = parsed
		}
	}

	offset := 0
	if o := r.URL.Query().Get("offset"); o != "" {
		if parsed, err := strconv.Atoi(o); err == nil {
			offset = parsed
		}
	}

	query := r.URL.Query().Get("query")

	spans, err := GetAllRootSpans(db, query, limit, offset)
	if err != nil {
		http.Error(w, "Failed to fetch traces", http.StatusInternalServerError)
		return
	}

	w.Header().Set("Content-Type", "application/json")
	json.NewEncoder(w).Encode(spans)
}
