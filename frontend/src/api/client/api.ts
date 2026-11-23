import { Api } from '../generated/api.gen';

/**
 * The API client. Frontend and API share the origin, and schemaf's session
 * cookie authenticates every call: a signed-in user, a Guest or nobody.
 *
 * Every response is read as JSON: the generated client reads a body only for
 * calls that answer one, so the error body ({error, code}) of a call without
 * a response body (e.g. POST /api/updates/check, DELETE /api/users/{id})
 * would go unread, and its code with it. A successful empty answer stays
 * empty.
 */
export const api = new Api<unknown>({
  baseApiParams: { credentials: 'same-origin', format: 'json' },
}).api;
