/**
 * Invite and re-login codes are German words joined by "-" (server: functional/wordtoken).
 * Invites have 3 words; re-login codes have 4 and are stored with PARTICIPANT_TOKEN_PREFIX.
 */

export const PARTICIPANT_TOKEN_PREFIX = "participant-";
export const INVITE_WORD_COUNT = 3;
export const PARTICIPANT_WORD_COUNT = 4;

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

/** Old base64 tokens carry the prefix already and must pass through untouched. */
export function toParticipantToken(tokenOrWords: string): string {
  if (tokenOrWords.startsWith(PARTICIPANT_TOKEN_PREFIX)) return tokenOrWords;
  return PARTICIPANT_TOKEN_PREFIX + normalizeWordToken(tokenOrWords);
}

const WORD_CODE = /^[a-z]+(-[a-z]+)+$/;

/** The part of a token people read aloud, or null for old base64 tokens. */
export function speakableCode(token: string): string | null {
  const code = token.startsWith(PARTICIPANT_TOKEN_PREFIX)
    ? token.slice(PARTICIPANT_TOKEN_PREFIX.length)
    : token;
  return WORD_CODE.test(code) ? code : null;
}

export function inviteLinkPath(inviteToken: string): string {
  return `/invites/${inviteToken}/accept`;
}

/** Word tokens drop the prefix in the URL; the login page restores it. */
export function participantLinkPath(participantToken: string): string {
  return `/invites/participant/${speakableCode(participantToken) ?? participantToken}`;
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
export function participantShareLinkPath(participantToken: string): string {
  return wordLinkPath(
    participantToken,
    PARTICIPANT_WORD_COUNT,
    participantLinkPath,
  );
}
