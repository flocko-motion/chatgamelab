/**
 * How the playback-engine comes to a session: opened with a session ID it
 * resumes that one; opened with `new` and a game or a share token it creates
 * one. Headless, so the node harness starts sessions the way the page does.
 */
import type { CreateRequest, CreateResponse, Refusal } from "./protocol.js";
import { explain } from "./state.js";
import { notice, type Notice } from "./texts.js";

/** What `new` names: a game, or a share token. */
export interface Selector {
  readonly kind: "game" | "share";
  readonly id: string;
}

/** The little the playback-engine keeps, such as browser storage; never throws. */
export interface Memory {
  get(key: string): string | null;
  set(key: string, value: string): void;
  remove(key: string): void;
  keys(): string[];
}

/**
 * Where the playback-engine keeps its sessions: `sessions` across visits, so a
 * reloaded host page can offer to continue; `pending` for one tab, so a reload
 * mid-creation repeats its one-time key.
 */
export interface Memories {
  readonly sessions: Memory;
  readonly pending: Memory;
}

/** What the page was opened for. */
export type Opening =
  | { readonly kind: "resume"; readonly id: string; readonly selector: Selector | null }
  | { readonly kind: "new"; readonly selector: Selector }
  | { readonly kind: "nothing" };

export function opening(params: URLSearchParams, memories: Memories): Opening {
  const id = params.get("session");
  if (id) return { kind: "resume", id, selector: selectorOf(memories.sessions, id) };
  if (params.has("new")) {
    const game = params.get("game");
    const share = params.get("share");
    if (game) return { kind: "new", selector: { kind: "game", id: game } };
    if (share) return { kind: "new", selector: { kind: "share", id: share } };
  }
  return { kind: "nothing" };
}

const SESSION_PREFIX = "cgl.session.";
const ONCE_PREFIX = "cgl.once.";

function slot(selector: Selector): string {
  return `${selector.kind}:${selector.id}`;
}

/** The session this browser last played for a game or share, if any. */
export function remembered(memory: Memory, selector: Selector): string | null {
  return memory.get(SESSION_PREFIX + slot(selector));
}

export function remember(memory: Memory, selector: Selector, id: string): void {
  memory.set(SESSION_PREFIX + slot(selector), id);
}

export function forget(memory: Memory, selector: Selector): void {
  memory.remove(SESSION_PREFIX + slot(selector));
}

/** Which game or share a remembered session was for, so it can start anew. */
export function selectorOf(memory: Memory, id: string): Selector | null {
  for (const key of memory.keys()) {
    if (!key.startsWith(SESSION_PREFIX) || memory.get(key) !== id) continue;
    const [kind, ...rest] = key.slice(SESSION_PREFIX.length).split(":");
    if (kind === "game" || kind === "share") return { kind, id: rest.join(":") };
  }
  return null;
}

/** A session, or what to tell the player about why there is none. */
export type Created = { readonly id: string } | { readonly notice: Notice };

/**
 * Creates a session at the engine's root. The one-time key is kept until the
 * session is remembered, so a reload while creating answers the same session;
 * a caller may name the key, as a test repeating a creation does.
 */
export async function create(
  root: URL,
  selector: Selector,
  language: string,
  memories: Memories,
  key?: string,
): Promise<Created> {
  const onceKey = ONCE_PREFIX + slot(selector);
  const once = key ?? memories.pending.get(onceKey) ?? randomKey();
  memories.pending.set(onceKey, once);

  const body: CreateRequest = { [selector.kind]: selector.id, language, once };
  let response: Response;
  try {
    response = await fetch(new URL("sessions", root), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify(body),
    });
  } catch {
    return { notice: notice("errors.serverUnreachable") };
  }
  if (!response.ok) {
    // Refused: the next try is a new intent, with a new key.
    memories.pending.remove(onceKey);
    return { notice: explain(await readRefusal(response)) };
  }
  const created = (await response.json()) as CreateResponse;
  remember(memories.sessions, selector, created.id);
  memories.pending.remove(onceKey);
  return { id: created.id };
}

/** Reads a transport's error body, which is a refusal unless something else answered. */
export async function readRefusal(response: Response): Promise<Refusal> {
  const text = await response.text();
  try {
    const parsed = JSON.parse(text) as Partial<Refusal>;
    if (typeof parsed.problem === "string") {
      return { problem: parsed.problem, message: parsed.message ?? "" };
    }
  } catch {
    // Not a refusal: a proxy or the mounting server answered.
  }
  return { problem: "failed", message: text.trim() || `The game server answered ${response.status}.` };
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
