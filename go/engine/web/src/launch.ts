/**
 * How the playback-engine comes to a session: opened with a session ID it
 * plays that one; opened with `new` and a game or a share token it creates
 * one, as the standalone dev server opens it. On the platform the web app
 * creates and opens sessions itself, so the playback-engine keeps no memory
 * of sessions. Headless, so the node harness starts sessions the way the
 * page does.
 */
import { refusalOf, type CreateRequest, type EngineClient, type Mode } from "./client.js";
import { explain } from "./state.js";
import { notice, type Notice } from "./texts.js";

/**
 * What `new` names: a game, or a share token. A share embedded on a verified
 * page also carries the host's ID of its visitor and of this request, which
 * the host is asked about before the game starts. A mode, live or rounds, is
 * the wish for a game that plays both, as an embed may name it; without one
 * the player is asked.
 */
export interface Selector {
  readonly kind: "game" | "share";
  readonly id: string;
  readonly player?: string;
  readonly request?: string;
  readonly mode?: Mode;
}

const MODES: readonly Mode[] = ["live", "rounds"];

/** The mode a page names, if it names one there is. */
function modeOf(params: URLSearchParams): { mode?: Mode } {
  const mode = params.get("mode");
  return MODES.find((known) => known === mode) ? { mode: mode as Mode } : {};
}

/** The little the playback-engine keeps, such as browser storage; never throws. */
export interface Memory {
  get(key: string): string | null;
  set(key: string, value: string): void;
  remove(key: string): void;
  keys(): string[];
}

/** What the page was opened for. */
export type Opening =
  | { readonly kind: "resume"; readonly id: string }
  | { readonly kind: "new"; readonly selector: Selector }
  | { readonly kind: "nothing" };

export function opening(params: URLSearchParams): Opening {
  const id = params.get("session");
  if (id) return { kind: "resume", id };
  if (params.has("new")) {
    const game = params.get("game");
    const share = params.get("share");
    if (game) return { kind: "new", selector: { kind: "game", id: game, ...modeOf(params) } };
    if (share) {
      const player = params.get("player");
      const request = params.get("request");
      const host = { ...(player === null ? {} : { player }), ...(request === null ? {} : { request }) };
      return { kind: "new", selector: { kind: "share", id: share, ...hostIds(host), ...modeOf(params) } };
    }
  }
  return { kind: "nothing" };
}

const ONCE_PREFIX = "cgl.once.";

/**
 * A session; the player to choose live or rounds, as the game plays both;
 * or what to tell the player about why there is none.
 */
export type Created = { readonly id: string } | { readonly choose: true } | { readonly notice: Notice };

/**
 * Creates a session through the engine's client. The one-time key is kept in
 * pending, for one tab, until the session exists, so a reload while creating
 * answers the same session; a caller may name the key, as a test repeating a
 * creation does.
 */
export async function create(
  client: EngineClient,
  selector: Selector,
  language: string,
  pending: Memory,
  key?: string,
): Promise<Created> {
  const onceKey = `${ONCE_PREFIX}${selector.kind}:${selector.id}`;
  const once = key ?? pending.get(onceKey) ?? randomKey();
  pending.set(onceKey, once);

  const mode = selector.mode ? { mode: selector.mode } : {};
  const body: CreateRequest =
    selector.kind === "game"
      ? { game: selector.id, language, once, ...mode }
      : { share: selector.id, ...hostIds(selector), language, once, ...mode };
  try {
    const { data } = await client.postApiEngineSessions(body);
    pending.remove(onceKey);
    return { id: data.id };
  } catch (thrown) {
    const refusal = await refusalOf(thrown);
    if (!refusal) return { notice: notice("errors.serverUnreachable") };
    // The same intent goes on once the player has chosen, with the same key.
    if (refusal.problem === "choose_mode") return { choose: true };
    // Refused: the next try is a new intent, with a new key.
    pending.remove(onceKey);
    return { notice: explain(refusal) };
  }
}

/** The host's IDs a share carries: both, once either is named, or none. */
function hostIds(ids: { player?: string; request?: string }): { player?: string; request?: string } {
  return ids.player !== undefined || ids.request !== undefined
    ? { player: ids.player ?? "", request: ids.request ?? "" }
    : {};
}

function randomKey(): string {
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}

/** A memory that lasts as long as the process, for node and for a browser that keeps nothing. */
export function transientMemory(): Memory {
  const held = new Map<string, string>();
  return {
    get: (key) => held.get(key) ?? null,
    set: (key, value) => void held.set(key, value),
    remove: (key) => void held.delete(key),
    keys: () => [...held.keys()],
  };
}
