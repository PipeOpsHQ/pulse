/**
 * Ensures a URL starts with https:// if it's a domain/URL but missing a protocol.
 * @param {string} url - The URL to check.
 * @returns {string} - The URL with https:// prepended if needed.
 */
export function ensureHttps(url) {
  if (!url || typeof url !== 'string') return url;

  const trimmed = url.trim();
  if (!trimmed) return trimmed;

  // If it already has a protocol, leave it alone
  if (trimmed.includes('://')) {
    return trimmed;
  }

  // If it looks like a domain (has a dot) or is just a string that should be a URL
  // We'll be opinionated and prepend https://
  return `https://${trimmed}`;
}
