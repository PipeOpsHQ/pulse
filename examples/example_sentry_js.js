// Example: Using Sentry JavaScript SDK with this alternative
// Install: npm install @sentry/browser

import * as Sentry from "@sentry/browser";

// Get your DSN from the project detail page in the UI
// Format: http://{api_key}@{host}/{project_id}
Sentry.init({
  dsn: "http://YOUR_API_KEY@localhost:8080/YOUR_PROJECT_ID",
  environment: "production",
  release: "1.0.0",
  tracesSampleRate: 1.0, // Set to 0.0 to disable performance monitoring
});

// Test error capture
try {
  throw new Error("Test error from Sentry SDK");
} catch (error) {
  Sentry.captureException(error);
}

// Or manually capture a message
Sentry.captureMessage("Something went wrong", "error");