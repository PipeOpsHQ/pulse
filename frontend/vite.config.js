import { defineConfig } from 'vite';
import { svelte } from '@sveltejs/vite-plugin-svelte';
import { copyFileSync, existsSync } from 'fs';
import { join } from 'path';

// Plugin to preserve favicon during build
const preserveFaviconPlugin = () => {
  let faviconPath = null;
  return {
    name: 'preserve-favicon',
    buildStart() {
      // Save favicon path before build
      const path = join(process.cwd(), 'public', 'favicon.svg');
      if (existsSync(path)) {
        faviconPath = path;
      }
    },
    writeBundle() {
      // Restore favicon after build if it was deleted
      if (faviconPath && existsSync(faviconPath)) {
        const dest = join(process.cwd(), 'public', 'favicon.svg');
        if (!existsSync(dest)) {
          copyFileSync(faviconPath, dest);
        }
      }
    }
  };
};

export default defineConfig({
  plugins: [svelte(), preserveFaviconPlugin()],
  server: {
    port: 5173,
    proxy: {
      '/api': {
        target: 'http://localhost:8080',
        changeOrigin: true
      }
    }
  },
  build: {
    outDir: 'public',
    emptyOutDir: true
  }
});
