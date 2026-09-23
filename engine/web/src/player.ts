/**
 * The headless player: a full game is playable through this class alone, with
 * no DOM and no audio device. It owns the session state and the turn timeline;
 * anything drawing it is a pure function of `state`.
 *
 * Integration tests drive this directly against a running engine.
 */
import {
  isSessionEvent,
  type BlockState,
  type SessionEvent,
  type UsageReport,
} from "./protocol.js";
import type { Snapshot, Topology } from "./state.js";
import { emptyState, type PlayerState } from "./state.js";
import type { Transport, TransportMessage } from "./transport.js";

export type Listener = (state: PlayerState) => void;

export class Player {
  readonly state: PlayerState = emptyState();

  #listeners = new Set<Listener>();
  #nextUtterance = 1;
  #open: Utterance | null = null;

  constructor(private readonly transport: Transport) {}

  /**
   * Notified after every state change, so a view never polls.
   *
   * `immediate` is opt-out because a waiter must not be called before its own
   * unsubscribe handle exists.
   */
  subscribe(listener: Listener, immediate = true): () => void {
    this.#listeners.add(listener);
    if (immediate) listener(this.state);
    return () => this.#listeners.delete(listener);
  }

  connect(): void {
    this.#set("connecting");
    this.transport.open({
      onOpen: () => this.#set("open"),
      onClose: () => this.#set("closed"),
      onError: () => this.#set("failed"),
      onMessage: (data) => this.#receive(data),
    });
  }

  close(): void {
    this.transport.close();
    this.#set("closed");
  }

  /**
   * Sends one frame of player speech. The player's own words are never
   * transcribed, so nothing about this appears in the state.
   */
  speak(pcm: ArrayBuffer): void {
    this.transport.send(pcm);
  }

  /**
   * Sends typed input. A live session takes both typed and spoken words, and a
   * turn-based genre takes only these.
   */
  say(text: string): void {
    this.transport.send(JSON.stringify({ text }));
  }

  /** Marks the player taking the floor, so the next reply starts a new line. */
  takeFloor(): void {
    if (this.#open) {
      this.#open.complete = true;
      this.#open = null;
      this.#notify();
    }
  }

  /** Resolves once the connection is open, and rejects if it never gets there. */
  async whenReady(timeoutMs = 5_000): Promise<void> {
    await this.until((state) => {
      if (state.connection === "failed" || state.connection === "closed") {
        throw new Error(`session ${state.connection}`);
      }
      return state.connection === "open";
    }, timeoutMs);
  }

  /** Resolves when the state satisfies a predicate, which is how tests wait. */
  until(predicate: (state: PlayerState) => boolean, timeoutMs = 10_000): Promise<PlayerState> {
    if (predicate(this.state)) return Promise.resolve(this.state);

    return new Promise((resolve, reject) => {
      let stop = (): void => {};
      const settle = (run: () => void): void => {
        clearTimeout(timer);
        stop();
        run();
      };

      const timer = setTimeout(
        () => settle(() => reject(new Error("timed out waiting for the expected state"))),
        timeoutMs,
      );

      stop = this.subscribe((state) => {
        let matched = false;
        try {
          matched = predicate(state);
        } catch (error) {
          settle(() => reject(error));
          return;
        }
        if (matched) settle(() => resolve(state));
      }, false);
    });
  }

  #receive(data: TransportMessage): void {
    if (data instanceof ArrayBuffer) {
      this.state.audioChunks++;
      this.#notify();
      return;
    }

    let parsed: unknown;
    try {
      parsed = JSON.parse(data);
    } catch {
      return;
    }
    if (!isSessionEvent(parsed)) return;
    this.#apply(parsed);
  }

  #apply(event: SessionEvent): void {
    switch (event.Stream) {
      case "text":
        // An empty delta closes the utterance rather than adding to it: the
        // boundary rides the same stream as the text so that it is ordered
        // against it.
        if (event.Value === "") this.takeFloor();
        else this.#extend(event.Value);
        break;
      case "flag":
        this.state.flags.push(event.Value);
        // What follows a flag is the steered reply, not more of what was
        // flagged, so the line ends here.
        this.takeFloor();
        break;
      case "image":
        this.state.image = event.Value;
        break;
      case "props":
        this.state.props = parseProps(event.Value);
        break;
      case "state":
        this.#applyBlockState(event.Value);
        break;
      case "usage":
        this.#applyUsage(event.Value);
        break;
      default:
        this.state.notes.push(`${event.Stream}: ${event.Value}`);
    }
    this.#notify();
  }

  /**
   * Rebuilds everything a reloaded page missed: the wiring, where the session
   * stands, and the conversation so far. The socket carries what is happening;
   * this carries what has happened.
   */
  async restore(base: string | URL): Promise<void> {
    await this.loadTopology(base);
    await this.#loadSnapshot(base);
    await this.#loadHistory(base);
  }

  async #loadSnapshot(base: string | URL): Promise<void> {
    const response = await fetch(new URL("state", base));
    if (!response.ok) return;
    const snapshot = (await response.json()) as Snapshot;
    this.state.phases = snapshot.phases;
    this.state.props = snapshot.props;
    this.state.usage = snapshot.usage;
    this.state.started = snapshot.started;
    this.#notify();
  }

  /**
   * Replayed events go through exactly the same path as live ones, so there is
   * no second way to render a conversation and no second place for it to be
   * wrong.
   */
  async #loadHistory(base: string | URL): Promise<void> {
    const response = await fetch(new URL("history", base));
    if (!response.ok) return;
    for (const event of (await response.json()) as SessionEvent[]) {
      this.#apply(event);
    }
  }

  /**
   * Fetches the wiring once. It does not change during a session, so it is a
   * plain GET rather than something the live stream has to carry.
   */
  async loadTopology(base: string | URL): Promise<void> {
    const response = await fetch(new URL("topology", base));
    if (!response.ok) return;
    this.state.topology = (await response.json()) as Topology;
    // A gate that opened before the wiring arrived would otherwise leave the
    // game looking unstarted.
    this.#recomputeStarted();
    this.#notify();
  }

  /**
   * The game has begun once the gate reports ready. The gate is found by its
   * role rather than by name, so the core does not have to know what a genre
   * calls it.
   */
  #recomputeStarted(): void {
    const gate = this.state.topology?.nodes.find((node) => node.role === "gate");
    this.state.started = gate ? this.state.phases[gate.name] === "ready" : false;
  }

  #applyBlockState(value: string): void {
    let parsed: unknown;
    try {
      parsed = JSON.parse(value);
    } catch {
      return;
    }
    const report = parsed as Partial<BlockState>;
    if (typeof report.node !== "string" || typeof report.phase !== "string") return;

    this.state.phases[report.node] = report.phase;
    this.#recomputeStarted();
  }

  #applyUsage(value: string): void {
    try {
      this.state.usage = JSON.parse(value) as UsageReport;
    } catch {
      // A malformed report costs a display update, not the conversation.
    }
  }

  #extend(delta: string): void {
    if (!this.#open) {
      this.#open = { id: this.#nextUtterance++, speaker: "npc", text: "", complete: false };
      this.state.utterances.push(this.#open);
    }
    this.#open.text += delta;
  }

  #set(connection: PlayerState["connection"]): void {
    this.state.connection = connection;
    this.#notify();
  }

  #notify(): void {
    for (const listener of this.#listeners) listener(this.state);
  }
}

type Utterance = PlayerState["utterances"][number];

/**
 * Status arrives as Go's map formatting — `map[Health:Good Gold:12]` — because
 * the engine's props stream is a rendered map today. Parsed here so the state
 * carries real fields rather than a string the UI would have to pick apart.
 */
function parseProps(value: string): Record<string, string> {
  const inner = value.replace(/^map\[/, "").replace(/\]$/, "");
  const props: Record<string, string> = {};
  for (const pair of inner.split(" ")) {
    if (!pair) continue;
    const separator = pair.indexOf(":");
    if (separator < 0) continue;
    props[pair.slice(0, separator)] = pair.slice(separator + 1);
  }
  return props;
}
