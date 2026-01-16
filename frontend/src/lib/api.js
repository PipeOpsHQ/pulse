import { get } from 'svelte/store';
import { token, logout } from '../stores/auth';

const API_BASE = '/api';

/**
 * Creates headers with authentication token if available
 */
function getHeaders(contentType = 'application/json') {
  const headers = {};

  if (contentType) {
    headers['Content-Type'] = contentType;
  }

  const currentToken = get(token);
  if (currentToken) {
    headers['Authorization'] = `Bearer ${currentToken}`;
  }

  return headers;
}

/**
 * Handles API response, including 401 redirects
 */
async function handleResponse(response) {
  if (response.status === 401) {
    // Token expired or invalid, logout and redirect
    logout();
    if (typeof window !== 'undefined') {
      window.location.href = '/login';
    }
    const error = new Error('Unauthorized');
    error.statusCode = 401;
    throw error;
  }

  if (!response.ok) {
    const errorText = await response.text();
    const error = new Error(errorText || `HTTP ${response.status}`);
    error.statusCode = response.status;
    throw error;
  }

  // Handle empty responses
  const text = await response.text();
  if (!text) return null;

  try {
    return JSON.parse(text);
  } catch {
    return text;
  }
}

/**
 * GET request
 */
export async function apiGet(endpoint) {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    method: 'GET',
    headers: getHeaders(),
  });
  return handleResponse(response);
}

/**
 * POST request
 */
export async function apiPost(endpoint, data) {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    method: 'POST',
    headers: getHeaders(),
    body: JSON.stringify(data),
  });
  return handleResponse(response);
}

/**
 * PUT request
 */
export async function apiPut(endpoint, data) {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    method: 'PUT',
    headers: getHeaders(),
    body: JSON.stringify(data),
  });
  return handleResponse(response);
}

/**
 * PATCH request
 */
export async function apiPatch(endpoint, data) {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    method: 'PATCH',
    headers: getHeaders(),
    body: JSON.stringify(data),
  });
  return handleResponse(response);
}

/**
 * DELETE request
 */
export async function apiDelete(endpoint) {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    method: 'DELETE',
    headers: getHeaders(),
  });
  return handleResponse(response);
}

// Convenience object for cleaner imports
export const api = {
  get: apiGet,
  post: apiPost,
  put: apiPut,
  patch: apiPatch,
  delete: apiDelete,
};

export default api;
