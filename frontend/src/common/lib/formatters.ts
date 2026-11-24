/**
 * Shared formatting utilities for dates, times, and other values.
 */

/**
 * Formats a date string as a relative time in the given language
 * (e.g. "now", "5 min. ago", "vor 2 Std.", "3 days ago").
 *
 * @param dateString - ISO date string or undefined
 * @param locale - The UI language (i18n.language)
 * @returns Formatted relative time string, or empty string if no date provided
 */
export function formatRelativeTime(dateString: string | undefined, locale: string): string {
  if (!dateString) return '';

  const diffSeconds = Math.round((new Date(dateString).getTime() - Date.now()) / 1000);
  const format = new Intl.RelativeTimeFormat(locale, { numeric: 'auto', style: 'short' });
  const abs = Math.abs(diffSeconds);
  if (abs < 60) return format.format(0, 'second');
  if (abs < 3600) return format.format(Math.round(diffSeconds / 60), 'minute');
  if (abs < 86400) return format.format(Math.round(diffSeconds / 3600), 'hour');
  if (abs < 86400 * 30) return format.format(Math.round(diffSeconds / 86400), 'day');
  if (abs < 86400 * 365) return format.format(Math.round(diffSeconds / (86400 * 30)), 'month');
  return format.format(Math.round(diffSeconds / (86400 * 365)), 'year');
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
