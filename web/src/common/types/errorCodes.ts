/**
 * Centralized backend error codes.
 * These match the error codes defined in the Go backend.
 */
export const ErrorCodes = {
  GENERIC: 'error',
  VALIDATION: 'validation_error',
  UNAUTHORIZED: 'unauthorized',
  FORBIDDEN: 'forbidden',
  NOT_FOUND: 'not_found',
  CONFLICT: 'conflict',
  INVALID_PLATFORM: 'invalid_platform',
  INVALID_INPUT: 'invalid_input',
  SERVER_ERROR: 'server_error',
  USER_NOT_REGISTERED: 'user_not_registered',
  // AI-related error codes
  INVALID_API_KEY: 'invalid_api_key',
  ORG_VERIFICATION_REQUIRED: 'organization_verification_required',
  BILLING_NOT_ACTIVE: 'billing_not_active',
  RATE_LIMIT_EXCEEDED: 'rate_limit_exceeded',
  INSUFFICIENT_QUOTA: 'insufficient_quota',
  CONTENT_FILTERED: 'content_filtered',
  AI_ERROR: 'ai_error',
} as const;

export type ErrorCode = (typeof ErrorCodes)[keyof typeof ErrorCodes];

/**
 * Check if a value is a known error code.
 */
export function isKnownErrorCode(code: string): code is ErrorCode {
  return Object.values(ErrorCodes).includes(code as ErrorCode);
}

/**
 * Extract error code from various error shapes.
 * Handles API client HttpResponse.error, direct error objects, and nested patterns.
 * Note: HttpResponse extends fetch Response which has a `type` property (e.g., "cors")
 * that we must NOT use - we only want the API error's code/type.
 */
export function extractErrorCode(error: unknown): ErrorCode | null {
  if (!error || typeof error !== 'object') return null;

  // API client HttpResponse pattern: { error: { code, message, type } }
  // This is the primary pattern for API errors
  if ('error' in error && error.error && typeof error.error === 'object') {
    const nested = error.error as Record<string, unknown>;
    if ('code' in nested && typeof nested.code === 'string') {
      return isKnownErrorCode(nested.code) ? nested.code : null;
    }
    if ('type' in nested && typeof nested.type === 'string') {
      return isKnownErrorCode(nested.type) ? nested.type : null;
    }
  }

  // Direct code property (for plain error objects, not HttpResponse)
  if ('code' in error && typeof error.code === 'string') {
    return isKnownErrorCode(error.code) ? error.code : null;
  }

  // Skip direct `type` property - HttpResponse.type is "cors"/"basic" etc, not an error code

  return null;
}

/**
 * Extract any error code string from an error, even if not known.
 * Used for including unknown codes in fallback messages.
 * Note: Skips Response.type which can be "cors"/"basic" etc.
 */
export function extractRawErrorCode(error: unknown): string | null {
  if (!error || typeof error !== 'object') return null;

  // API client HttpResponse pattern: { error: { code, message, type } }
  if ('error' in error && error.error && typeof error.error === 'object') {
    const nested = error.error as Record<string, unknown>;
    if ('code' in nested && typeof nested.code === 'string') {
      return nested.code;
    }
    if ('type' in nested && typeof nested.type === 'string') {
      return nested.type;
    }
  }

  // Direct code property only (skip type - Response.type is "cors" etc)
  if ('code' in error && typeof error.code === 'string') {
    return error.code;
  }

  return null;
}

