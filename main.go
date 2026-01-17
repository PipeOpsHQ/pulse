package main

import (
	"log"
	"net/http"
	"os"
	"strconv"
	"strings"
	"time"

	"github.com/getsentry/sentry-go"
	sentryhttp "github.com/getsentry/sentry-go/http"
	"github.com/gorilla/mux"
	"github.com/joho/godotenv"
	_ "github.com/mattn/go-sqlite3"
)

// Types are defined in database.go

type ErrorResponse struct {
	ID string `json:"id"`
}

func main() {
	// Load .env file if it exists
	if err := godotenv.Load(); err != nil {
		log.Println("No .env file found or error loading it, using system environment variables")
	}

	// Initialize Sentry
	if dsn := os.Getenv("SENTRY_DSN"); dsn != "" {
		err := sentry.Init(sentry.ClientOptions{
			Dsn:              dsn,
			Environment:      getEnvOrDefault("SENTRY_ENVIRONMENT", "development"),
			TracesSampleRate: getFloatEnvOrDefault("SENTRY_TRACES_SAMPLE_RATE", 1.0),
			BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
				// Add custom filtering or modification here if needed
				return event
			},
		})
		if err != nil {
			log.Printf("Sentry initialization failed: %v", err)
		} else {
			log.Println("Sentry initialized successfully")
			defer sentry.Flush(2 * time.Second)
		}
	} else {
		log.Println("Sentry DSN not configured, skipping initialization")
	}

	db, err := InitDB()
	if err != nil {
		log.Fatal("Failed to initialize database:", err)
	}
	defer db.Close()

	r := mux.NewRouter()

	// Sentry middleware for request tracking
	sentryMiddleware := sentryhttp.New(sentryhttp.Options{
		Repanic: true,
	})
	r.Use(sentryMiddleware.Handle)

	// CORS middleware
	r.Use(corsMiddleware)

	// Public routes
	r.HandleFunc("/api/health", healthCheck).Methods("GET", "OPTIONS")

	// API routes
	api := r.PathPrefix("/api").Subrouter()

	// Apply Auth Middleware to all API routes
	// Note: The middleware itself handles exclusions for login and ingestion endpoints
	api.Use(AuthMiddleware)

	// Auth routes
	api.HandleFunc("/auth/login", func(w http.ResponseWriter, r *http.Request) {
		handleLogin(w, r, db)
	}).Methods("POST", "OPTIONS")

	api.HandleFunc("/auth/mfa/verify", func(w http.ResponseWriter, r *http.Request) {
		handleMFAVerify(w, r, db)
	}).Methods("POST", "OPTIONS")

	api.HandleFunc("/auth/me", func(w http.ResponseWriter, r *http.Request) {
		handleMe(w, r, db)
	}).Methods("GET", "OPTIONS")

	// Sentry-compatible error ingestion endpoint: /api/{project_id}/store/
	api.HandleFunc("/{projectId}/store/", func(w http.ResponseWriter, r *http.Request) {
		storeErrorSentry(w, r, db)
	}).Methods("POST", "OPTIONS")

	// Sentry-compatible endpoint without trailing slash (some SDKs don't use it)
	api.HandleFunc("/{projectId}/store", func(w http.ResponseWriter, r *http.Request) {
		storeErrorSentry(w, r, db)
	}).Methods("POST", "OPTIONS")

	// Sentry project discovery endpoint (used by some SDKs)
	api.HandleFunc("/{projectId}/", func(w http.ResponseWriter, r *http.Request) {
		getProjectDiscovery(w, r, db)
	}).Methods("GET", "OPTIONS")

	// Legacy endpoint for backward compatibility
	api.HandleFunc("/store", func(w http.ResponseWriter, r *http.Request) {
		storeError(w, r, db)
	}).Methods("POST", "OPTIONS")

	// Project management
	api.HandleFunc("/projects", func(w http.ResponseWriter, r *http.Request) {
		getProjects(w, r, db)
	}).Methods("GET", "OPTIONS")

	api.HandleFunc("/projects", func(w http.ResponseWriter, r *http.Request) {
		createProject(w, r, db)
	}).Methods("POST", "OPTIONS")

	api.HandleFunc("/projects/{id}", func(w http.ResponseWriter, r *http.Request) {
		getProject(w, r, db)
	}).Methods("GET", "OPTIONS")

	api.HandleFunc("/projects/{id}", func(w http.ResponseWriter, r *http.Request) {
		deleteProject(w, r, db)
	}).Methods("DELETE", "OPTIONS")

	api.HandleFunc("/projects/{id}/quota", func(w http.ResponseWriter, r *http.Request) {
		updateProjectQuota(w, r, db)
	}).Methods("PATCH", "OPTIONS")

	api.HandleFunc("/projects/{id}/settings", func(w http.ResponseWriter, r *http.Request) {
		getProjectSettings(w, r, db)
	}).Methods("GET", "OPTIONS")

	api.HandleFunc("/projects/{id}/settings", func(w http.ResponseWriter, r *http.Request) {
		updateProjectSettings(w, r, db)
	}).Methods("PUT", "PATCH", "OPTIONS")

	api.HandleFunc("/errors", func(w http.ResponseWriter, r *http.Request) {
		getErrors(w, r, db)
	}).Methods("GET", "OPTIONS")

	// Error retrieval
	api.HandleFunc("/projects/{projectId}/errors", func(w http.ResponseWriter, r *http.Request) {
		getProjectErrors(w, r, db)
	}).Methods("GET", "OPTIONS")

	// Sentry Envelope Endpoint
	api.HandleFunc("/{projectId}/envelope/", func(w http.ResponseWriter, r *http.Request) {
		handleEnvelopeSentry(w, r, db)
	}).Methods("POST", "OPTIONS")

	api.HandleFunc("/{projectId}/envelope", func(w http.ResponseWriter, r *http.Request) {
		handleEnvelopeSentry(w, r, db)
	}).Methods("POST", "OPTIONS")

	// Compatibility aliases for legacy/test paths
	api.HandleFunc("/projects/{projectId}/envelope", func(w http.ResponseWriter, r *http.Request) {
		handleEnvelopeSentry(w, r, db)
	}).Methods("POST", "OPTIONS")

	api.HandleFunc("/projects/{projectId}/envelope/", func(w http.ResponseWriter, r *http.Request) {
		handleEnvelopeSentry(w, r, db)
	}).Methods("POST", "OPTIONS")

	// Root-level aliases for SDKs that might omit /api (if configured that way)
	r.HandleFunc("/{projectId}/store/", func(w http.ResponseWriter, r *http.Request) {
		storeErrorSentry(w, r, db)
	}).Methods("POST", "OPTIONS")
	r.HandleFunc("/{projectId}/store", func(w http.ResponseWriter, r *http.Request) {
		storeErrorSentry(w, r, db)
	}).Methods("POST", "OPTIONS")
	r.HandleFunc("/{projectId}/envelope/", func(w http.ResponseWriter, r *http.Request) {
		handleEnvelopeSentry(w, r, db)
	}).Methods("POST", "OPTIONS")
	r.HandleFunc("/{projectId}/envelope", func(w http.ResponseWriter, r *http.Request) {
		handleEnvelopeSentry(w, r, db)
	}).Methods("POST", "OPTIONS")
	r.HandleFunc("/{projectId}/", func(w http.ResponseWriter, r *http.Request) {
		getProjectDiscovery(w, r, db)
	}).Methods("GET", "OPTIONS")

	// Trace retrieval
	api.HandleFunc("/projects/{projectId}/traces", func(w http.ResponseWriter, r *http.Request) {
		getProjectTraces(w, r, db)
	}).Methods("GET", "OPTIONS")

	// Uptime Monitors
	api.HandleFunc("/projects/{id}/monitors", func(w http.ResponseWriter, r *http.Request) {
		getProjectMonitors(w, r, db)
	}).Methods("GET", "OPTIONS")

	api.HandleFunc("/projects/{id}/monitors", func(w http.ResponseWriter, r *http.Request) {
		createMonitor(w, r, db)
	}).Methods("POST", "OPTIONS")

	api.HandleFunc("/projects/{id}/monitors/{monitorId}", func(w http.ResponseWriter, r *http.Request) {
		updateMonitor(w, r, db)
	}).Methods("PUT", "PATCH", "OPTIONS")

	api.HandleFunc("/projects/{id}/monitors/{monitorId}", func(w http.ResponseWriter, r *http.Request) {
		deleteMonitor(w, r, db)
	}).Methods("DELETE", "OPTIONS")

	// Public status page API endpoint (no auth required)
	r.HandleFunc("/api/status/{projectId}", func(w http.ResponseWriter, r *http.Request) {
		getStatusPage(w, r, db)
	}).Methods("GET", "OPTIONS")

	api.HandleFunc("/insights", func(w http.ResponseWriter, r *http.Request) {
		getInsights(w, r, db)
	}).Methods("GET", "OPTIONS")

	api.HandleFunc("/admin/stats", func(w http.ResponseWriter, r *http.Request) {
		getSystemStats(w, r, db)
	}).Methods("GET", "OPTIONS")

	api.HandleFunc("/projects/{projectId}/traces/{traceId}", func(w http.ResponseWriter, r *http.Request) {
		getTraceDetails(w, r, db)
	}).Methods("GET", "OPTIONS")

	api.HandleFunc("/errors/{id}", func(w http.ResponseWriter, r *http.Request) {
		getError(w, r, db)
	}).Methods("GET", "OPTIONS")
	api.HandleFunc("/errors/{id}/occurrences", func(w http.ResponseWriter, r *http.Request) {
		getErrorOccurrences(w, r, db)
	}).Methods("GET", "OPTIONS")

	api.HandleFunc("/errors/{id}", func(w http.ResponseWriter, r *http.Request) {
		deleteError(w, r, db)
	}).Methods("DELETE", "OPTIONS")

	api.HandleFunc("/errors/{id}", func(w http.ResponseWriter, r *http.Request) {
		updateErrorStatus(w, r, db)
	}).Methods("PATCH", "OPTIONS")

	// Coverage routes - support both /api/projects/{projectId}/coverage and /api/{projectId}/coverage
	api.HandleFunc("/projects/{projectId}/coverage", func(w http.ResponseWriter, r *http.Request) {
		uploadProjectCoverage(w, r, db)
	}).Methods("POST", "OPTIONS")

	// Shorter route for coverage upload (API key auth only)
	api.HandleFunc("/{projectId}/coverage", func(w http.ResponseWriter, r *http.Request) {
		uploadProjectCoverage(w, r, db)
	}).Methods("POST", "OPTIONS")

	api.HandleFunc("/projects/{projectId}/coverage/history", func(w http.ResponseWriter, r *http.Request) {
		getProjectCoverageHistory(w, r, db)
	}).Methods("GET", "OPTIONS")

	api.HandleFunc("/projects/{projectId}/coverage/badge", func(w http.ResponseWriter, r *http.Request) {
		getCoverageBadge(w, r, db)
	}).Methods("GET", "OPTIONS")

	api.HandleFunc("/projects/{projectId}/coverage/snapshots/{snapshotId}/files", func(w http.ResponseWriter, r *http.Request) {
		getProjectFileCoverage(w, r, db)
	}).Methods("GET", "OPTIONS")

	// Security Vault
	api.HandleFunc("/security/mfa/setup", func(w http.ResponseWriter, r *http.Request) {
		setupMFA(w, r, db)
	}).Methods("POST", "OPTIONS")

	api.HandleFunc("/security/mfa/enable", func(w http.ResponseWriter, r *http.Request) {
		enableMFA(w, r, db)
	}).Methods("POST", "OPTIONS")

	api.HandleFunc("/projects/{id}/rotate-key", func(w http.ResponseWriter, r *http.Request) {
		rotateProjectAPIKey(w, r, db)
	}).Methods("POST", "OPTIONS")

	api.HandleFunc("/projects/{id}/key-history", func(w http.ResponseWriter, r *http.Request) {
		getProjectAPIKeyHistory(w, r, db)
	}).Methods("GET", "OPTIONS")

	api.HandleFunc("/projects/{id}/security-policies", func(w http.ResponseWriter, r *http.Request) {
		getSecurityPolicies(w, r, db)
	}).Methods("GET", "OPTIONS")

	api.HandleFunc("/projects/{id}/security-policies", func(w http.ResponseWriter, r *http.Request) {
		updateSecurityPolicies(w, r, db)
	}).Methods("POST", "OPTIONS")

	// Global Settings
	api.HandleFunc("/settings", func(w http.ResponseWriter, r *http.Request) {
		getSettings(w, r, db)
	}).Methods("GET", "OPTIONS")

	api.HandleFunc("/settings", func(w http.ResponseWriter, r *http.Request) {
		updateSettings(w, r, db)
	}).Methods("POST", "PATCH", "OPTIONS")

	api.HandleFunc("/system/cleanup", func(w http.ResponseWriter, r *http.Request) {
		runCleanup(w, r, db)
	}).Methods("POST", "OPTIONS")

	// Serve static files (SPA routing - serve index.html for all non-API routes)
	r.PathPrefix("/assets/").Handler(http.StripPrefix("/assets/", http.FileServer(http.Dir("./frontend/dist/assets"))))

	r.PathPrefix("/").HandlerFunc(func(w http.ResponseWriter, req *http.Request) {
		// Don't serve index.html for API routes
		if strings.HasPrefix(req.URL.Path, "/api") {
			http.NotFound(w, req)
			return
		}

		// Try to serve the file directly if it exists in dist
		path := "./frontend/dist" + req.URL.Path
		if info, err := os.Stat(path); err == nil && !info.IsDir() {
			http.FileServer(http.Dir("./frontend/dist")).ServeHTTP(w, req)
			return
		}

		// Otherwise serve index.html for SPA routing
		http.ServeFile(w, req, "./frontend/dist/index.html")
	})

	port := os.Getenv("PORT")
	if port == "" {
		port = "8080"
	}

	// Start background workers
	go StartMonitorWorker(db)

	log.Printf("Pulse OSS starting on port %s", port)
	log.Fatal(http.ListenAndServe(":"+port, r))
}

func corsMiddleware(next http.Handler) http.Handler {
	return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
		w.Header().Set("Access-Control-Allow-Origin", "*")
		w.Header().Set("Access-Control-Allow-Methods", "GET, POST, PUT, DELETE, OPTIONS")
		w.Header().Set("Access-Control-Allow-Headers", "Content-Type, Authorization, X-Sentry-Auth")

		if r.Method == "OPTIONS" {
			w.WriteHeader(http.StatusOK)
			return
		}

		next.ServeHTTP(w, r)
	})
}

// getEnvOrDefault returns the environment variable value or a default
func getEnvOrDefault(key, defaultValue string) string {
	if value := os.Getenv(key); value != "" {
		return value
	}
	return defaultValue
}

// getFloatEnvOrDefault returns the environment variable as float64 or a default
func getFloatEnvOrDefault(key string, defaultValue float64) float64 {
	if value := os.Getenv(key); value != "" {
		if parsed, err := strconv.ParseFloat(value, 64); err == nil {
			return parsed
		}
	}
	return defaultValue
}
