import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';

// Vite configuration
export default defineConfig({
  plugins: [svelte()],
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
    emptyOutDir: true
  }
});
