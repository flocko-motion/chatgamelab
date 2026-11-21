import { Api } from '../generated/api.gen';

/**
 * The API client. Frontend and API share the origin, and schemaf's session
 * cookie authenticates every call: a signed-in user, a Guest or nobody.
 */
export const api = new Api<unknown>({
  baseApiParams: { credentials: 'same-origin' },
}).api;
