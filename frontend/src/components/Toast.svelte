<script>
  import { toast } from '../stores/toast';
  import { fade, fly } from 'svelte/transition';

  function getIcon(type) {
    switch (type) {
      case 'success': return '✓';
      case 'error': return '✕';
      case 'warning': return '!';
      case 'info': return 'ℹ';
      case 'unauthorized': return 'LOCK';
      case 'forbidden': return 'X';
      case 'notfound': return '?';
      case 'ratelimited': return '!';
      case 'servererror': return 'X';
      default: return 'i';
    }
  }
</script>

<div class="toast-container">
  {#each $toast as notification (notification.id)}
    <div
      class="toast {notification.type}"
      in:fly={{ x: 300, duration: 300 }}
      out:fade={{ duration: 200 }}
    >
      <div class="toast-icon">{getIcon(notification.type)}</div>
      <div class="toast-message">{notification.message}</div>
      <button class="toast-close" on:click={() => toast.remove(notification.id)}>×</button>
    </div>
  {/each}
</div>

<style>
  .toast-container {
    position: fixed;
    top: 1rem;
    right: 1rem;
    z-index: 9999;
    display: flex;
    flex-direction: column;
    gap: 0.75rem;
    max-width: 400px;
  }

  .toast {
    display: flex;
    align-items: center;
    gap: 0.75rem;
    padding: 1rem 1.25rem;
    border-radius: 10px;
    background-color: var(--bg-card);
    border: 1px solid var(--border-color);
    box-shadow: 0 8px 24px rgba(0, 0, 0, 0.4);
    backdrop-filter: blur(12px);
  }

  .toast.success {
    border-left: 4px solid #10b981;
  }

  .toast.error {
    border-left: 4px solid #ef4444;
  }

  .toast.warning {
    border-left: 4px solid #f59e0b;
  }

  .toast.info {
    border-left: 4px solid #3b82f6;
  }

  .toast.unauthorized {
    border-left: 4px solid #ef4444;
    background: linear-gradient(90deg, rgba(239, 68, 68, 0.1) 0%, var(--bg-card) 100%);
  }

  .toast.forbidden {
    border-left: 4px solid #ef4444;
    background: linear-gradient(90deg, rgba(239, 68, 68, 0.1) 0%, var(--bg-card) 100%);
  }

  .toast.notfound {
    border-left: 4px solid #f59e0b;
    background: linear-gradient(90deg, rgba(245, 158, 11, 0.1) 0%, var(--bg-card) 100%);
  }

  .toast.ratelimited {
    border-left: 4px solid #f97316;
    background: linear-gradient(90deg, rgba(249, 115, 22, 0.1) 0%, var(--bg-card) 100%);
  }

  .toast.servererror {
    border-left: 4px solid #ef4444;
    background: linear-gradient(90deg, rgba(239, 68, 68, 0.15) 0%, var(--bg-card) 100%);
  }

  .toast-icon {
    width: 24px;
    height: 24px;
    border-radius: 50%;
    display: flex;
    align-items: center;
    justify-content: center;
    font-size: 0.75rem;
    font-weight: 700;
    flex-shrink: 0;
  }

  .toast.success .toast-icon {
    background-color: rgba(16, 185, 129, 0.2);
    color: #10b981;
  }

  .toast.error .toast-icon {
    background-color: rgba(239, 68, 68, 0.2);
    color: #ef4444;
  }

  .toast.warning .toast-icon {
    background-color: rgba(245, 158, 11, 0.2);
    color: #f59e0b;
  }

  .toast.info .toast-icon {
    background-color: rgba(59, 130, 246, 0.2);
    color: #3b82f6;
  }

  .toast.unauthorized .toast-icon,
  .toast.forbidden .toast-icon {
    background-color: rgba(239, 68, 68, 0.25);
    color: #ef4444;
  }

  .toast.notfound .toast-icon {
    background-color: rgba(245, 158, 11, 0.25);
    color: #f59e0b;
  }

  .toast.ratelimited .toast-icon {
    background-color: rgba(249, 115, 22, 0.25);
    color: #f97316;
  }

  .toast.servererror .toast-icon {
    background-color: rgba(239, 68, 68, 0.3);
    color: #ef4444;
  }

  .toast-message {
    flex: 1;
    color: var(--text-primary);
    font-size: 0.875rem;
    line-height: 1.4;
  }

  .toast-close {
    background: transparent;
    border: none;
    color: var(--text-secondary);
    font-size: 1.25rem;
    cursor: pointer;
    padding: 0;
    line-height: 1;
    opacity: 0.6;
    transition: opacity 0.2s;
  }

  .toast-close:hover {
    opacity: 1;
  }
</style>
