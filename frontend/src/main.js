import { mount } from 'svelte';
import App from './App.svelte';
import './app.css';
import { initSentry } from './lib/sentry.js';

// Initialize Sentry for error tracking and performance monitoring
initSentry();

const target = document.getElementById('app');
if (!target) {
  throw new Error('Target element #app not found');
}

const app = mount(App, {
  target,
});

export default app;