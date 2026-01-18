import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { sentryVitePlugin } from '@sentry/vite-plugin';

// Vite configuration
export default defineConfig({
  plugins: [
    svelte(),
    // Sentry plugin for source map upload (only in production builds)
    ...(process.env.NODE_ENV === 'production' && process.env.SENTRY_AUTH_TOKEN
      ? [
          sentryVitePlugin({
            org: process.env.SENTRY_ORG,
            project: process.env.SENTRY_PROJECT,
            authToken: process.env.SENTRY_AUTH_TOKEN,
            sourcemaps: {
              assets: './dist/**',
              ignore: ['node_modules'],
            },
          }),
        ]
      : []),
  ],
  publicDir: 'public',
  server: {
    port: 5173,
    open: false,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true
      }
    }
  },
  build: {
    outDir: 'dist',
    emptyOutDir: true,
    // Enable source maps for better error context
    sourcemap: true,
  }
});
