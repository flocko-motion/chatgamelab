/**
 * Calls a raw endpoint (a file up or down) that the generated client cannot
 * carry. A failure throws what the generated client throws, the response
 * with its error body ({error, code}) as `error`, so error handling reads
 * both alike. Same origin, so schemaf's session cookie authenticates it.
 */
export async function rawRequest(path: string, init: RequestInit = {}): Promise<Response> {
  const response = await fetch(path, { credentials: "same-origin", ...init });
  if (!response.ok) {
    const failed = response as Response & { error: unknown };
    failed.error = await response
      .clone()
      .json()
      .catch(() => ({ error: response.statusText, code: "error" }));
    throw failed;
  }
  return response;
}
