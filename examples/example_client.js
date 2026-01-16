// Example client for sending errors to Pulse
// Usage: node example_client.js YOUR_API_KEY

const apiKey = process.argv[2] || 'YOUR_API_KEY';
const apiUrl = process.env.API_URL || 'http://localhost:8080';

async function sendError() {
  const errorData = {
    message: 'Example error from Node.js client',
    level: 'error',
    environment: 'development',
    platform: 'node',
    release: '1.0.0',
    stacktrace: {
      frames: [
        {
          filename: 'example.js',
          lineno: 42,
          function: 'doSomething',
          code: 'throw new Error("Something went wrong");'
        }
      ]
    },
    context: {
      os: {
        name: 'Linux',
        version: '5.4.0'
      },
      runtime: {
        name: 'node',
        version: '18.0.0'
      }
    },
    user: {
      id: '123',
      username: 'testuser',
      email: 'test@example.com'
    },
    tags: {
      component: 'api',
      feature: 'authentication'
    }
  };

  try {
    const response = await fetch(`${apiUrl}/api/store`, {
      method: 'POST',
      headers: {
        'Content-Type': 'application/json',
        'X-Sentry-Auth': `Sentry sentry_key=${apiKey}, sentry_version=7`
      },
      body: JSON.stringify(errorData)
    });

    if (response.ok) {
      const result = await response.json();
      console.log('Error sent successfully!', result);
    } else {
      console.error('Failed to send error:', response.status, await response.text());
    }
  } catch (error) {
    console.error('Error sending request:', error);
  }
}

sendError();