/**
 * Shared formatting utilities for dates, times, and other values.
 */

/**
 * Formats a date string as a relative time (e.g., "just now", "5m ago", "2h ago", "3d ago")
 * 
 * @param dateString - ISO date string or undefined
 * @returns Formatted relative time string, or empty string if no date provided
 */
export function formatRelativeTime(dateString?: string): string {
  if (!dateString) return '';
  
  const date = new Date(dateString);
  const now = new Date();
  const diffMs = now.getTime() - date.getTime();
  const diffMins = Math.floor(diffMs / 60000);
  const diffHours = Math.floor(diffMs / 3600000);
  const diffDays = Math.floor(diffMs / 86400000);

  if (diffMins < 1) return 'just now';
  if (diffMins < 60) return `${diffMins}m ago`;
  if (diffHours < 24) return `${diffHours}h ago`;
  return `${diffDays}d ago`;
}

/**
 * Formats a date string as a localized date (e.g., "Jan 15, 2024")
 * 
 * @param dateString - ISO date string or undefined
 * @param locale - Locale string (defaults to browser locale)
 * @returns Formatted date string, or empty string if no date provided
 */
export function formatDate(dateString?: string, locale?: string): string {
  if (!dateString) return '';
  
  const date = new Date(dateString);
  return date.toLocaleDateString(locale, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
  });
}

/**
 * Formats a date string as a localized date and time (e.g., "Jan 15, 2024, 2:30 PM")
 * 
 * @param dateString - ISO date string or undefined
 * @param locale - Locale string (defaults to browser locale)
 * @returns Formatted date and time string, or empty string if no date provided
 */
export function formatDateTime(dateString?: string, locale?: string): string {
  if (!dateString) return '';
  
  const date = new Date(dateString);
  return date.toLocaleDateString(locale, {
    year: 'numeric',
    month: 'short',
    day: 'numeric',
    hour: 'numeric',
    minute: '2-digit',
  });
}

/**
 * Formats a date string as time only (e.g., "2:30 PM")
 * 
 * @param dateString - ISO date string or undefined
 * @param locale - Locale string (defaults to browser locale)
 * @returns Formatted time string, or empty string if no date provided
 */
export function formatTime(dateString?: string, locale?: string): string {
  if (!dateString) return '';
  
  const date = new Date(dateString);
  return date.toLocaleTimeString(locale, {
    hour: 'numeric',
    minute: '2-digit',
  });
}
