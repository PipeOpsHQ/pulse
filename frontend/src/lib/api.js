import { get } from 'svelte/store';
import { token, logout } from '../stores/auth';

const API_BASE = '/api';

// Cache for GET requests
const cache = new Map();
const CACHE_TTL = 5000; // 5 seconds default TTL
const pendingRequests = new Map(); // Request deduplication

/**
 * Generate cache key from endpoint
 */
function getCacheKey(endpoint) {
  return `GET:${endpoint}`;
}

/**
 * Check if cache entry is still valid
 */
function isCacheValid(entry) {
  if (!entry) return false;
  return Date.now() - entry.timestamp < entry.ttl;
}

/**
 * Clear cache for an endpoint (call after mutations)
 */
export function clearCache(endpoint) {
  if (endpoint) {
    const key = getCacheKey(endpoint);
    cache.delete(key);
    // Also clear related caches (e.g., clearing /errors clears /errors?status=...)
    for (const [k] of cache) {
      if (k.includes(endpoint.split('?')[0])) {
        cache.delete(k);
      }
    }
  } else {
    cache.clear();
  }
}

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
 * GET request with caching and request deduplication
 */
export async function apiGet(endpoint, options = {}) {
  const cacheKey = getCacheKey(endpoint);
  const ttl = options.ttl !== undefined ? options.ttl : CACHE_TTL;
  const useCache = options.cache !== false; // Default to true

  // Check cache first
  if (useCache) {
    const cached = cache.get(cacheKey);
    if (isCacheValid(cached)) {
      return cached.data;
    }
  }

  // Check if there's a pending request for this endpoint
  if (pendingRequests.has(cacheKey)) {
    // Wait for the pending request to complete
    return pendingRequests.get(cacheKey);
  }

  // Create new request
  const requestPromise = (async () => {
    try {
      const response = await fetch(`${API_BASE}${endpoint}`, {
        method: 'GET',
        headers: getHeaders(),
      });
      const data = await handleResponse(response);

      // Cache the response
      if (useCache) {
        cache.set(cacheKey, {
          data,
          timestamp: Date.now(),
          ttl,
        });
      }

      return data;
    } finally {
      // Remove from pending requests after completion
      pendingRequests.delete(cacheKey);
    }
  })();

  // Store pending request
  pendingRequests.set(cacheKey, requestPromise);

  return requestPromise;
}

/**
 * POST request (clears related cache)
 */
export async function apiPost(endpoint, data) {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    method: 'POST',
    headers: getHeaders(),
    body: JSON.stringify(data),
  });
  const result = await handleResponse(response);

  // Clear related cache
  clearCache(endpoint);

  return result;
}

/**
 * PUT request (clears related cache)
 */
export async function apiPut(endpoint, data) {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    method: 'PUT',
    headers: getHeaders(),
    body: JSON.stringify(data),
  });
  const result = await handleResponse(response);

  // Clear related cache
  clearCache(endpoint);

  return result;
}

/**
 * PATCH request (clears related cache)
 */
export async function apiPatch(endpoint, data) {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    method: 'PATCH',
    headers: getHeaders(),
    body: JSON.stringify(data),
  });
  const result = await handleResponse(response);

  // Clear related cache
  clearCache(endpoint);

  return result;
}

/**
 * DELETE request (clears related cache)
 */
export async function apiDelete(endpoint) {
  const response = await fetch(`${API_BASE}${endpoint}`, {
    method: 'DELETE',
    headers: getHeaders(),
  });
  const result = await handleResponse(response);

  // Clear related cache
  clearCache(endpoint);

  return result;
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
