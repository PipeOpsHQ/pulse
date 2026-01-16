import { writable } from 'svelte/store';
import { getToastTypeFromStatus } from '../lib/statusColors';

function createToastStore() {
  const { subscribe, update } = writable([]);

  let id = 0;

  function add(message, type = 'info', duration = 5000) {
    const toastId = ++id;

    update(toasts => [...toasts, { id: toastId, message, type }]);

    if (duration > 0) {
      setTimeout(() => {
        remove(toastId);
      }, duration);
    }

    return toastId;
  }

  function remove(toastId) {
    update(toasts => toasts.filter(t => t.id !== toastId));
  }

  function fromHttpError(error, duration = 5000) {
    const statusCode = error?.statusCode || error?.status || 500;
    let type = getToastTypeFromStatus(statusCode);

    // Map specific status codes to specific toast types
    if (statusCode === 401) type = 'unauthorized';
    else if (statusCode === 403) type = 'forbidden';
    else if (statusCode === 404) type = 'notfound';
    else if (statusCode === 429) type = 'ratelimited';
    else if (statusCode >= 500) type = 'servererror';

    const message = error?.message || `HTTP ${statusCode} Error`;
    return add(message, type, duration);
  }

  return {
    subscribe,
    add,
    success: (message, duration) => add(message, 'success', duration),
    error: (message, duration) => add(message, 'error', duration),
    warning: (message, duration) => add(message, 'warning', duration),
    info: (message, duration) => add(message, 'info', duration),
    fromHttpError,
    remove
  };
}

export const toast = createToastStore();
