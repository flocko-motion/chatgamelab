import { Api, type RequestParams } from '../generated/api.gen';

/**
 * The API client. Frontend and API share the origin, and schemaf's session
 * cookie authenticates every call: a signed-in user, a participant or nobody.
 */
export const api = new Api<unknown>({
  baseApiParams: { credentials: 'same-origin' },
}).api;

/**
 * Request params carrying query parameters. The generated client declares no
 * query parameters until schemaf#58 lands; then they become method arguments
 * and this goes.
 */
export function withQuery(
  query: Record<string, string | number | boolean | undefined>,
): RequestParams {
  return { query } as RequestParams;
}
