/**
 * Invite and re-login codes are German words joined by "-" (server: functional/wordtoken).
 * Invites have 3 words; re-login codes have 4 and are stored with GUEST_TOKEN_PREFIX, the server's token prefix.
 */

export const GUEST_TOKEN_PREFIX = "guest-";
/** The prefix of codes issued before the role was called Guest; they still sign in. */
const LEGACY_GUEST_TOKEN_PREFIX = "participant-";

/** The token's prefix, new or legacy, or "" when it has none. */
function guestTokenPrefix(token: string): string {
  if (token.startsWith(GUEST_TOKEN_PREFIX)) return GUEST_TOKEN_PREFIX;
  if (token.startsWith(LEGACY_GUEST_TOKEN_PREFIX)) return LEGACY_GUEST_TOKEN_PREFIX;
  return "";
}

export const INVITE_WORD_COUNT = 3;
export const GUEST_WORD_COUNT = 4;

const TRANSLITERATION: Record<string, string> = {
  ä: "ae",
  ö: "oe",
  ü: "ue",
  ß: "ss",
};

/** Mirrors wordtoken.Normalize: lowercase, umlauts transliterated, separators collapsed to "-". */
export function normalizeWordToken(input: string): string {
  return input
    .trim()
    .toLowerCase()
    .replace(/[äöüß]/g, (c) => TRANSLITERATION[c])
    .split(/[\s._-]+/)
    .filter(Boolean)
    .join("-");
}

export function wordCount(normalized: string): number {
  return normalized ? normalized.split("-").length : 0;
}

/**
 * What a Guest signs in with: a whole token (old base64 tokens included) passes
 * through untouched; words are sent normalized and without a prefix, so the
 * server can match codes issued under either prefix.
 */
export function toGuestToken(tokenOrWords: string): string {
  if (guestTokenPrefix(tokenOrWords)) return tokenOrWords;
  return normalizeWordToken(tokenOrWords);
}

const WORD_CODE = /^[a-z]+(-[a-z]+)+$/;

/** The part of a token people read aloud, or null for old base64 tokens. */
export function speakableCode(token: string): string | null {
  const code = token.slice(guestTokenPrefix(token).length);
  return WORD_CODE.test(code) ? code : null;
}

export function inviteLinkPath(inviteToken: string): string {
  return `/invites/${inviteToken}/accept`;
}

/** Word tokens drop the prefix in the URL; the login page restores it. */
export function guestLinkPath(guestToken: string): string {
  return `/invites/guest/${speakableCode(guestToken) ?? guestToken}`;
}

export const CODE_LINK_PREFIX = "/code/";

function wordLinkPath(
  token: string,
  words: number,
  fallback: (token: string) => string,
): string {
  const code = speakableCode(token);
  return code && wordCount(code) === words
    ? CODE_LINK_PREFIX + code
    : fallback(token);
}

/** /code/<three words>; old ws-… tokens keep their accept link. */
export function inviteShareLinkPath(inviteToken: string): string {
  return wordLinkPath(inviteToken, INVITE_WORD_COUNT, inviteLinkPath);
}

/** /code/<four words>; old long tokens keep their re-login link. */
export function guestShareLinkPath(guestToken: string): string {
  return wordLinkPath(
    guestToken,
    GUEST_WORD_COUNT,
    guestLinkPath,
  );
}
