import * as Sentry from "@sentry/svelte";

/**
 * Initialize Sentry for error tracking and performance monitoring
 * DSN should be configured via VITE_SENTRY_DSN environment variable
 */
export function initSentry() {
  const dsn = import.meta.env.VITE_SENTRY_DSN;

  // Only initialize if DSN is configured
  if (!dsn) {
    console.warn('[Sentry] DSN not configured. Skipping initialization.');
    return;
  }

  Sentry.init({
    dsn,
    environment: import.meta.env.VITE_SENTRY_ENVIRONMENT || import.meta.env.MODE || 'development',

    // Performance Monitoring
    tracesSampleRate: parseFloat(import.meta.env.VITE_SENTRY_TRACES_SAMPLE_RATE || '1.0'),

    // Ensure stack traces include source context
    attachStacktrace: true,

    // Session Replay (optional, can be disabled)
    replaysSessionSampleRate: 0.1, // 10% of sessions
    replaysOnErrorSampleRate: 1.0, // 100% of sessions with errors

    integrations: [
      Sentry.browserTracingIntegration(),
      Sentry.replayIntegration({
        maskAllText: false,
        blockAllMedia: false,
      }),
    ],

    // Filter out non-error console logs
    beforeSend(event, hint) {
      // Don't send events for known non-issues
      if (event.exception) {
        const error = hint.originalException;
        if (error && typeof error === 'object' && 'message' in error) {
          const message = String(error.message);
          // Filter out common non-issues
          if (message.includes('ResizeObserver loop')) {
            return null;
          }
        }
      }
      return event;
    },
  });

  console.log('[Sentry] Initialized successfully');
}
