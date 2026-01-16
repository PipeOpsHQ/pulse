# Example: Using Sentry Python SDK with this alternative
# Install: pip install sentry-sdk

import sentry_sdk

# Get your DSN from the project detail page in the UI
# Format: http://{api_key}@{host}/{project_id}
sentry_sdk.init(
    dsn="http://YOUR_API_KEY@localhost:8080/YOUR_PROJECT_ID",
    environment="production",
    release="1.0.0",
    traces_sample_rate=1.0,  # Set to 0.0 to disable performance monitoring
)

# Test error capture
try:
    raise ValueError("Test error from Sentry SDK")
except Exception as e:
    sentry_sdk.capture_exception(e)

# Or manually capture a message
sentry_sdk.capture_message("Something went wrong", level="error")