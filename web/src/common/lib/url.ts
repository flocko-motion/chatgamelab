import { config } from '@/config/env';

/**
 * Get the base URL for the application.
 * Uses PUBLIC_URL from config if it's a full URL, otherwise combines with window.location.origin.
 * This should be used for all share links and external URLs.
 */
export function getBaseUrl(): string {
  const publicUrl = config.PUBLIC_URL;
  if (publicUrl && publicUrl.startsWith('http')) {
    return publicUrl;
  }
  if (typeof window !== 'undefined') {
    return publicUrl === '/' 
      ? window.location.origin 
      : `${window.location.origin}${publicUrl}`;
  }
  return publicUrl || '/';
}

/**
 * Build a full URL for sharing, using the proper PUBLIC_URL base.
 * @param path - The path to append (should start with /)
 */
export function buildShareUrl(path: string): string {
  const base = getBaseUrl();
  // Ensure no double slashes
  if (base.endsWith('/') && path.startsWith('/')) {
    return `${base}${path.slice(1)}`;
  }
  return `${base}${path}`;
}

/**
 * Get the cookie path from API_BASE_URL.
 * Extracts the path component (e.g., "/cgldev/api" from "https://example.com/cgldev/api").
 * Defaults to "/api" if not set.
 */
export function getCookiePath(): string {
  const apiBaseUrl = config.API_BASE_URL;
  if (!apiBaseUrl) {
    return '/api';
  }

  try {
    const parsed = new URL(apiBaseUrl);
    if (parsed.pathname && parsed.pathname !== '/') {
      return parsed.pathname.replace(/\/$/, ''); // Remove trailing slash
    }
  } catch {
    // If it's just a path (starts with /), use it directly
    if (apiBaseUrl.startsWith('/')) {
      return apiBaseUrl.replace(/\/$/, '');
    }
  }

  return '/api';
}
