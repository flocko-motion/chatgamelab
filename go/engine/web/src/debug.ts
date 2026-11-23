/**
 * The developer's record of a session: what crossed the live connection and
 * what changed, the latest entries kept, so a session that got stuck can be
 * read back as it stands, along with what led up to it. It is set up only on
 * a dev server, before the live connection opens.
 */
import type { PlaybackEngine } from "./playback-engine.js";
import type { PlaybackEngineState } from "./state.js";
import { describe, type Notice } from "./texts.js";
import type { TappedTransport, TransportMessage, Direction } from "./transport.js";

/** How many entries the record keeps; older ones fall off its start. */
export const CAPACITY = 500;

export type EntryKind = Direction | "phase" | "run" | "connection" | "notice" | "error";

/** One thing that happened, and when. */
export interface Entry {
  readonly time: string;
  readonly kind: EntryKind;
  /** A text frame in full, or an error's message. */
  readonly text?: string;
  /** An audio frame's size. */
  readonly bytes?: number;
  /** The block a phase change or run is about. */
  readonly node?: string;
  readonly from?: string | undefined;
  readonly to?: string;
}

/** What a stuck session is read from: the playback engine's state and the record. */
export interface Snapshot {
  readonly state: PlaybackEngineState;
  readonly buffer: Entry[];
}

/** The notices a session can hold, by their place in the state. */
const NOTICES = ["problem", "fault"] as const;

export class DebugRecorder {
  #entries: Entry[] = [];
  /** Where the next entry goes once the record is full. */
  #next = 0;

  constructor(
    private readonly capacity = CAPACITY,
    private readonly clock: () => Date = () => new Date(),
  ) {}

  /** Records every frame on the transport and every change to the playback engine's state. */
  record(transport: TappedTransport, playback: PlaybackEngine): void {
    transport.listen((direction, data) => this.#frame(direction, data));
    this.#watch(playback);
  }

  /** Records an error the page ran into. */
  error(message: string): void {
    this.#add({ kind: "error", text: message });
  }

  /** The entries kept, oldest first. */
  entries(): Entry[] {
    return [...this.#entries.slice(this.#next), ...this.#entries.slice(0, this.#next)];
  }

  snapshot(playback: PlaybackEngine): Snapshot {
    return { state: playback.state, buffer: this.entries() };
  }

  #frame(kind: Direction, data: TransportMessage): void {
    if (typeof data === "string") this.#add({ kind, text: data });
    else this.#add({ kind, bytes: data.byteLength });
  }

  /** Compares each state with the one before and records what changed. */
  #watch(playback: PlaybackEngine): void {
    const state = playback.state;
    let connection = state.connection;
    let phases = { ...state.phases };
    const runs = new Map(state.runs.map((r) => [r.id, r.done]));
    const notices = new Map(NOTICES.map((field) => [field, said(state[field])]));

    playback.subscribe((now) => {
      if (now.connection !== connection) {
        this.#add({ kind: "connection", from: connection, to: now.connection });
        connection = now.connection;
      }
      for (const [node, phase] of Object.entries(now.phases)) {
        if (phases[node] !== phase) this.#add({ kind: "phase", node, from: phases[node], to: phase });
      }
      phases = { ...now.phases };
      for (const run of now.runs) {
        const done = runs.get(run.id);
        if (done === undefined) this.#add({ kind: "run", node: run.node, to: "started" });
        if (run.done && !done) this.#add({ kind: "run", node: run.node, to: "done" });
        runs.set(run.id, run.done);
      }
      // Runs the Insights feed dropped are dropped here too.
      if (runs.size > now.runs.length) {
        const kept = new Set(now.runs.map((r) => r.id));
        for (const id of runs.keys()) if (!kept.has(id)) runs.delete(id);
      }
      for (const field of NOTICES) {
        const text = said(now[field]);
        if (text !== notices.get(field)) {
          this.#add({ kind: "notice", to: `${field}: ${text ?? "cleared"}` });
          notices.set(field, text);
        }
      }
    }, false);
  }

  #add(entry: Omit<Entry, "time">): void {
    const kept: Entry = { time: this.clock().toISOString(), ...entry };
    if (this.#entries.length < this.capacity) {
      this.#entries.push(kept);
      return;
    }
    this.#entries[this.#next] = kept;
    this.#next = (this.#next + 1) % this.capacity;
  }
}

function said(notice: Notice | null): string | null {
  return notice ? describe(notice) : null;
}

/** What an evaluated expression came to: a value fit for JSON, or what it threw. */
export type Evaluation = { value: unknown } | { error: string };

const AsyncFunction = Object.getPrototypeOf(async function () {}).constructor as new (
  ...args: string[]
) => (...values: unknown[]) => Promise<unknown>;

/**
 * Runs a developer's expression with the names in scope, `await` allowed: an
 * expression gives its value, a block of statements what it returns.
 */
export async function evaluate(expr: string, scope: Record<string, unknown>): Promise<Evaluation> {
  const names = Object.keys(scope);
  let run;
  try {
    run = new AsyncFunction(...names, `return (${expr}\n);`);
  } catch {
    try {
      run = new AsyncFunction(...names, expr);
    } catch (error) {
      return { error: String(error) };
    }
  }
  try {
    return { value: plain(await run(...names.map((n) => scope[n])), []) };
  } catch (error) {
    return { error: error instanceof Error ? (error.stack ?? String(error)) : String(error) };
  }
}

/** A value as JSON can carry it, with buffers, maps, sets and cycles said in words. */
function plain(value: unknown, ancestors: object[]): unknown {
  if (value === undefined || typeof value === "function" || typeof value === "symbol") return null;
  if (typeof value === "bigint") return value.toString();
  if (typeof value !== "object" || value === null) return value;
  if (ancestors.includes(value)) return "[circular]";
  if (value instanceof ArrayBuffer) return `ArrayBuffer(${value.byteLength})`;
  if (ArrayBuffer.isView(value)) return `${value.constructor.name}(${value.byteLength})`;
  if (value instanceof Error) return value.stack ?? String(value);
  const inner = [...ancestors, value];
  if (value instanceof Map) return [...value].map((pair) => plain(pair, inner));
  if (value instanceof Set || Array.isArray(value)) return [...value].map((v) => plain(v, inner));
  if (typeof (value as { toJSON?: unknown }).toJSON === "function") {
    return plain((value as { toJSON(): unknown }).toJSON(), ancestors);
  }
  const out: Record<string, unknown> = {};
  for (const [key, field] of Object.entries(value)) out[key] = plain(field, inner);
  return out;
}
