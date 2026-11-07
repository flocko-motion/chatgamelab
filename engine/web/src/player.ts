/**
 * The headless player: holds all session state with no DOM or audio device, so
 * a full game plays from node.
 */
import {
  isSessionEvent,
  type BlockState,
  type Choice,
  type Options,
  type SessionEvent,
  type Settled,
  type UsageReport,
} from "./protocol.js";
import type { Snapshot, ToggleName, Topology } from "./state.js";
import { emptyState, type PlayerState } from "./state.js";
import type { Transport, TransportMessage } from "./transport.js";
import type { Voice } from "./voice.js";

export type Listener = (state: PlayerState) => void;

export class Player {
  readonly state: PlayerState = emptyState();

  #listeners = new Set<Listener>();
  #nextUtterance = 1;
  #open: Utterance | null = null;

  /** Optional, so a headless run can watch a conversation it cannot join. */
  constructor(
    private readonly transport: Transport,
    private readonly voice?: Voice,
  ) {}

  /** `immediate` is opt-out so a waiter is not called before its unsubscribe handle exists. */
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
      // The transport retries on its own, so a drop is a state to sit through.
      onError: () => this.#set("reconnecting"),
      onMessage: (data) => this.#receive(data),
    });
  }

  /** Clears the conversation before replaying the engine's history, so nothing is said twice. */
  async rebuild(base: string | URL): Promise<void> {
    this.state.utterances = [];
    this.state.notes = [];
    this.#open = null;
    this.#nextUtterance = 1;
    this.#notify();
    await this.restore(base);
  }

  close(): void {
    this.voice?.close();
    this.transport.close();
    this.#set("closed");
  }

  /** Lets the conversation go now; `pausing` stands until the engine confirms the pause. */
  async pause(base: string | URL): Promise<void> {
    if (this.state.pausing || this.state.paused) return;
    this.state.pausing = true;
    // Stops sending at once, while the control holds its transition until the engine confirms.
    this.voice?.setSending(false);
    this.#notify();

    try {
      const response = await fetch(new URL("pause", base), { method: "POST" });
      if (!response.ok) throw new Error(`the engine answered ${response.status}`);
    } catch (error) {
      // Nothing was let go, so the conversation is live again.
      this.state.pausing = false;
      this.voice?.setSending(true);
      this.state.problem = `Could not pause: ${error instanceof Error ? error.message : String(error)}`;
      this.#notify();
    }
  }

  /** Picks a paused conversation back up by sending a fresh offer. */
  resume(): void {
    if (!this.state.paused || this.state.resuming) return;
    this.state.resuming = true;
    void this.#openVoice();
  }

  /**
   * Sends the player's choice: an offer by its ID, and a tier and toggles if it is
   * customizable. A refused choice leaves the dialog up and says why.
   */
  async choose(base: string | URL, offer: string, choice?: Choice): Promise<boolean> {
    const accepted = await this.#post(base, "choose", { offer, choice }, "Could not start");
    if (accepted) await this.loadOptions(base);
    return accepted;
  }

  /** Switches a feature during play; off is always allowed, on within the offer. */
  async setToggle(base: string | URL, toggle: ToggleName, on: boolean): Promise<boolean> {
    return this.#post(base, "toggles", { toggle, on }, "Could not switch");
  }

  async #post(base: string | URL, path: string, body: unknown, failure: string): Promise<boolean> {
    try {
      const response = await fetch(new URL(path, base), {
        method: "POST",
        headers: { "Content-Type": "application/json" },
        body: JSON.stringify(body),
      });
      if (!response.ok) throw new Error((await response.text()).trim() || `the engine answered ${response.status}`);
      this.state.problem = null;
      this.#notify();
      return true;
    } catch (error) {
      this.state.problem = `${failure}: ${error instanceof Error ? error.message : String(error)}`;
      this.#notify();
      return false;
    }
  }

  /** Fetches what the player may choose from, and what the session settled on. */
  async loadOptions(base: string | URL): Promise<void> {
    const response = await fetch(new URL("options", base));
    if (!response.ok) return;
    const options = (await response.json()) as Options;
    this.state.options = options;
    this.state.choosing = options.choosing;
    this.state.settled = options.settled ?? null;
    this.#notify();
  }

  /** Sends one frame of player speech, which is never transcribed into the state. */
  speak(pcm: ArrayBuffer): void {
    this.transport.send(pcm);
  }

  /** Sends typed input, which live and turn-based genres both take. */
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
        // An empty delta closes the utterance, ordered against the text it ends.
        if (event.Value === "") this.takeFloor();
        else this.#extend(event.Value);
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
      case "connect":
        // Only the first invitation is acted on: every later one follows a pause, and answering
        // it would resume and re-bill a conversation nobody asked to resume. The press offers.
        if (!this.state.invited) void this.#openVoice();
        this.state.invited = true;
        break;
      case "settled":
        this.#applySettled(event.Value);
        break;
      case "pause":
        // Let go for silence or on request; the session itself stays.
        this.state.paused = true;
        this.state.pausing = false;
        this.state.voice = "idle";
        this.voice?.close();
        break;
      default:
        this.state.notes.push(`${event.Stream}: ${event.Value}`);
    }
    this.#notify();
  }

  /** A failure loses only the voice; transcript and portrait keep arriving over the stream. */
  async #openVoice(): Promise<void> {
    // One negotiation at a time: two offers in flight race and close each other's connection.
    if (!this.voice || this.state.voice === "connecting") {
      this.state.resuming = false;
      return;
    }
    this.state.voice = "connecting";
    this.state.problem = null;
    this.#notify();
    try {
      await this.voice.connect();
      // Paused clears only once connected; until then it is still the paused conversation.
      this.state.voice = "live";
      this.state.paused = false;
      this.state.problem = null;
    } catch (error) {
      this.state.voice = "failed";
      this.state.problem = error instanceof Error ? error.message : String(error);
    }
    this.state.resuming = false;
    this.#notify();
  }

  /** Rebuilds what a reloaded page missed; the socket carries only what is happening. */
  async restore(base: string | URL): Promise<void> {
    // The wiring is the reachability check; loadTopology says what went wrong.
    if (!(await this.loadTopology(base))) {
      this.#notify();
      return;
    }
    await this.#loadSnapshot(base);
    await this.loadOptions(base);
    await this.#loadHistory(base);

    // Audio opens here from where the session stands, so replay never resumes a pause.
    if (this.state.invited && !this.state.paused && this.state.voice === "idle") {
      void this.#openVoice();
    }
  }

  async #loadSnapshot(base: string | URL): Promise<void> {
    const response = await fetch(new URL("state", base));
    if (!response.ok) return;
    const snapshot = (await response.json()) as Snapshot;
    this.state.phases = snapshot.phases;
    this.state.props = snapshot.props;
    this.state.usage = snapshot.usage;
    this.state.started = snapshot.started;
    this.state.choosing = snapshot.choosing;
    this.state.settled = snapshot.settled ?? null;
    // The invitation predates a reloaded page, so it asks where the conversation stands.
    // "open" is a conversation this page lost, so it offers again.
    switch (snapshot.conversation) {
      case "awaiting":
      case "open":
        this.state.invited = true;
        break;
      case "paused":
        this.state.invited = true;
        this.state.paused = true;
        break;
    }
    this.#notify();
  }

  /** Replayed history goes through the same path as live events. */
  async #loadHistory(base: string | URL): Promise<void> {
    const response = await fetch(new URL("history", base));
    if (!response.ok) return;
    const history: unknown = await response.json();
    if (!Array.isArray(history)) return;
    for (const event of history as SessionEvent[]) {
      this.#apply(event);
    }
  }

  /** Fetches the wiring once, as it never changes during a session. */
  async loadTopology(base: string | URL): Promise<boolean> {
    let response: Response;
    try {
      response = await fetch(new URL("topology", base));
    } catch {
      // Only an unreachable server throws instead of answering.
      this.state.problem = "Cannot reach the game server. Is it still running?";
      return false;
    }

    if (response.status === 404) {
      // The server is up; the session is missing, likeliest from a link without one.
      this.state.problem =
        "No such session. The address needs ?session=<id> — use the link the server printed.";
      return false;
    }
    if (!response.ok) {
      this.state.problem = `The game server answered ${response.status}.`;
      return false;
    }

    this.state.problem = null;
    this.state.topology = (await response.json()) as Topology;
    // A gate that opened before the wiring arrived would otherwise look unstarted.
    this.#recomputeStarted();
    this.#notify();
    return true;
  }

  /** The gate is found by role, so the core need not know what a genre calls it. */
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

  #applySettled(value: string): void {
    try {
      this.state.settled = JSON.parse(value) as Settled;
      this.state.choosing = false;
    } catch {
      // A malformed report costs a display update, not the game.
    }
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

/** Parses the props stream's Go map formatting, `map[Health:Good Gold:12]`, into fields. */
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
