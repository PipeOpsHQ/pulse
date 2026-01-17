# Sentry Integration Setup Guide

This guide explains how to configure Sentry SDK integration for both frontend and backend error tracking and performance monitoring.

## Overview

Pulse uses Sentry SDK to track errors and performance for both:
- **Frontend** (Svelte application)
- **Backend** (Go API server)

This allows Pulse to dogfood its own observability platform.

## Prerequisites

1. A running Pulse instance
2. A Pulse project created for tracking Pulse itself (meta!)

## Configuration Steps

### 1. Create a Pulse Project for Dogfooding

1. Log into your Pulse instance
2. Create a new project (e.g., "Pulse Dogfood")
3. Copy the API key from the project settings
4. Note your Pulse instance URL

### 2. Configure Backend (Go)

Edit `.env` file and set the following variables:

```bash
# Sentry Configuration (Backend)
SENTRY_DSN=https://<api-key>@<your-pulse-domain>/api/<project-id>/store/
SENTRY_ENVIRONMENT=development  # or production, staging, etc.
SENTRY_TRACES_SAMPLE_RATE=1.0   # 1.0 = 100%, 0.1 = 10%
```

**DSN Format:**
```
https://<api-key>@<your-pulse-domain>/api/<project-id>/store/
```

Example:
```
https://abc123def456@pulse.example.com/api/550e8400-e29b-41d4-a716-446655440000/store/
```

### 3. Configure Frontend (Svelte)

Create `frontend/.env.local` file (copy from `.env.local.example`):

```bash
# Frontend Environment Variables
VITE_SENTRY_DSN=https://<api-key>@<your-pulse-domain>/api/<project-id>/store/
VITE_SENTRY_ENVIRONMENT=development
VITE_SENTRY_TRACES_SAMPLE_RATE=1.0
```

**Note:** You can use the same project for both frontend and backend, or create separate projects for better organization.

### 4. Restart the Application

```bash
# Restart backend
go run .

# Restart frontend (in another terminal)
cd frontend
npm run dev
```

## Verification

### Test Frontend Error Tracking

Open browser console and run:
```javascript
throw new Error('Test frontend error');
```

Check your Pulse project - you should see the error appear.

### Test Backend Error Tracking

The backend automatically captures panics and errors. You can test by:
1. Making an invalid API request
2. Checking the Pulse project for the error event

### Test Performance Monitoring

Navigate through the Pulse UI. You should see:
- Frontend page loads as transactions
- Backend API requests as transactions
- Detailed span breakdowns for each operation

## Troubleshooting

### "404 default backend" Error

This error occurs when:
1. The DSN is not configured (empty)
2. The DSN format is incorrect
3. The Pulse instance is not reachable

**Solution:**
- Verify DSN format matches: `https://<api-key>@<domain>/api/<project-id>/store/`
- Ensure the Pulse instance is running and accessible
- Check that the project ID and API key are correct

### No Events Appearing

1. **Check DSN configuration** - Ensure DSN is set in both `.env` and `frontend/.env.local`
2. **Verify sample rate** - If set too low (e.g., 0.01), events might not be captured
3. **Check console logs** - Look for Sentry initialization messages
4. **Verify network** - Check browser/server network requests for Sentry events

### Source Maps Not Working (Frontend)

Source maps are automatically uploaded when using the Vite plugin (to be configured). For now, errors will show minified stack traces in production builds.

## Environment-Specific Configuration

### Development
```bash
SENTRY_ENVIRONMENT=development
SENTRY_TRACES_SAMPLE_RATE=1.0  # Capture all transactions
```

### Production
```bash
SENTRY_ENVIRONMENT=production
SENTRY_TRACES_SAMPLE_RATE=0.1  # Capture 10% of transactions
```

### Staging
```bash
SENTRY_ENVIRONMENT=staging
SENTRY_TRACES_SAMPLE_RATE=0.5  # Capture 50% of transactions
```

## Advanced Configuration

### Custom Error Filtering (Backend)

Edit `main.go` - the `BeforeSend` callback:

```go
BeforeSend: func(event *sentry.Event, hint *sentry.EventHint) *sentry.Event {
    // Filter out specific errors
    if event.Message == "ignore this" {
        return nil  // Don't send to Sentry
    }
    return event
},
```

### Custom Error Filtering (Frontend)

Edit `frontend/src/lib/sentry.js` - the `beforeSend` callback:

```javascript
beforeSend(event, hint) {
    // Filter out specific errors
    if (event.exception) {
        const error = hint.originalException;
        if (error && error.message.includes('ignore')) {
            return null;  // Don't send to Sentry
        }
    }
    return event;
},
```

## Security Notes

1. **Never commit DSN to version control** - Use `.env` and `.env.local` files (already in `.gitignore`)
2. **Use environment-specific DSNs** - Different DSNs for dev/staging/prod
3. **Rotate API keys regularly** - Especially if exposed or compromised

## Performance Impact

- **Frontend**: Minimal impact (~10-20KB bundle size increase)
- **Backend**: Negligible impact (<1% CPU/memory overhead)
- **Sample Rate**: Adjust based on traffic volume to control costs

## Support

For issues with Sentry integration:
1. Check this guide's troubleshooting section
2. Review Sentry SDK documentation
3. Open an issue on the Pulse GitHub repository
