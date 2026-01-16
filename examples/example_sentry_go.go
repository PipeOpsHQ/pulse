// Example: Using Sentry Go SDK with this alternative
// Install: go get github.com/getsentry/sentry-go

package main

import (
	"errors"
	"time"

	"github.com/getsentry/sentry-go"
)

func main() {
	// Get your DSN from the project detail page in the UI
	// Format: http://{api_key}@{host}/{project_id}
	err := sentry.Init(sentry.ClientOptions{
		Dsn:              "http://YOUR_API_KEY@localhost:8080/YOUR_PROJECT_ID",
		Environment:      "production",
		Release:          "1.0.0",
		TracesSampleRate: 1.0, // Set to 0.0 to disable performance monitoring
	})
	if err != nil {
		panic(err)
	}

	// Flush buffered events before the program terminates
	defer sentry.Flush(2 * time.Second)

	// Test error capture
	sentry.CaptureException(errors.New("Test error from Sentry SDK"))

	// Or manually capture a message
	sentry.CaptureMessage("Something went wrong")
}
