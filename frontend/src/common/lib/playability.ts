/**
 * Why a game cannot be played: the engine's problem codes, as game views say
 * them (`playability`) and as the engine refuses a session. The texts that
 * explain them to a player are the playback-engine's, under
 * `playbackEngine.problems`; the short ones beside a Play button are
 * `common.play.unplayable.<code>`.
 */

/** Each problem with the key of its explanation in the playbackEngine namespace. */
const PROBLEM_TEXT = {
  no_key: "problems.noKey",
  genre: "problems.genre",
  no_plays: "problems.noPlays",
  key_rejected: "problems.keyRejected",
  no_credit: "problems.noCredit",
  unreachable: "problems.unreachable",
  stale: "problems.stale",
  unknown: "problems.unknown",
  denied: "problems.denied",
  paused: "problems.paused",
  failed: "problems.failed",
} as const;

export type Problem = keyof typeof PROBLEM_TEXT;

/** A problem code as the web app knows it; anything else is failed. */
export function problemOf(code: string | undefined): Problem {
  return code !== undefined && code in PROBLEM_TEXT
    ? (code as Problem)
    : "failed";
}

/** The key, in the playbackEngine namespace, of the text explaining a problem. */
export function problemText(problem: Problem): string {
  return PROBLEM_TEXT[problem];
}

/** Why the caller cannot start a game, from its playability; null when they can. */
export function unplayable(subject: {
  readonly playability?: string;
}): Problem | null {
  if (!subject.playability || subject.playability === "playable") return null;
  return problemOf(subject.playability);
}

/**
 * Whether a problem also stops the caller's own session from going on.
 * Playability is about starting anew; continuing runs on the session's
 * stored platform and needs no genre or new play, so only a game they may no
 * longer play, or a paused workshop, holds it too.
 */
export function blocksContinue(problem: Problem): boolean {
  return problem === "denied" || problem === "paused";
}

/**
 * What a failed call to the engine says: its problem, from the error body
 * the generated client throws the response with, or "offline" when nothing
 * answered.
 */
export async function problemFromError(
  thrown: unknown,
): Promise<Problem | "offline"> {
  if (!(thrown instanceof Response)) return "offline";
  let body = (thrown as Response & { error?: unknown }).error;
  if (!hasCode(body)) {
    try {
      body = await thrown.clone().json();
    } catch {
      // Not JSON: something other than the engine answered.
    }
  }
  return hasCode(body) ? problemOf(body.code) : "failed";
}

function hasCode(value: unknown): value is { code: string } {
  return (
    typeof value === "object" &&
    value !== null &&
    typeof (value as { code?: unknown }).code === "string"
  );
}
