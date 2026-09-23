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
import type { Voice } from "./voice.js";

export type Listener = (state: PlayerState) => void;

export class Player {
  readonly state: PlayerState = emptyState();

  #listeners = new Set<Listener>();
  #nextUtterance = 1;
  #open: Utterance | null = null;

  /**
   * The voice is optional because the core must stay playable without one. A
   * headless run watches a conversation it cannot join, which is the honest
   * limit of a voice genre with no audio device.
   */
  constructor(
    private readonly transport: Transport,
    private readonly voice?: Voice,
  ) {}

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
      // The transport retries on its own, so a drop is a state to sit through
      // rather than a failure to report.
      onError: () => this.#set("reconnecting"),
      onMessage: (data) => this.#receive(data),
    });
  }

  /**
   * Rebuilds the conversation after a reconnection, from what the engine kept.
   *
   * A dropped socket misses whatever was said while it was away, and the
   * engine's history is the only record of it. Replaying on top of what is
   * already here would say everything twice, so the conversation is cleared
   * first and rebuilt whole — through the same path a fresh page uses, which is
   * why there is no second way for it to be wrong.
   */
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

  /**
   * Lets the conversation go now, rather than waiting out the silence.
   *
   * The asking is recorded before the request is sent and stands until the
   * engine puts a pause on the stream. Closing a live session is a round trip
   * through the provider, and a control that shows nothing until it comes back
   * reads as a control that did nothing — so the wait is a state rather than a
   * gap.
   */
  async pause(base: string | URL): Promise<void> {
    if (this.state.pausing || this.state.paused) return;
    this.state.pausing = true;
    // Stopped being heard at once. The engine's answer is a round trip away,
    // and a microphone that keeps reaching the character until it comes back
    // is a microphone that ignored the press.
    this.voice?.setSending(false);
    this.#notify();

    try {
      const response = await fetch(new URL("pause", base), { method: "POST" });
      if (!response.ok) throw new Error(`the engine answered ${response.status}`);
    } catch (error) {
      // Nothing was let go, so the conversation is still live: the control
      // goes back to saying so, and the character can hear again.
      this.state.pausing = false;
      this.voice?.setSending(true);
      this.state.problem = `Could not pause: ${error instanceof Error ? error.message : String(error)}`;
      this.#notify();
    }
  }

  /**
   * Picks a paused conversation back up. The engine is waiting for an offer, so
   * this opens a connection exactly as the first invitation did — and the
   * character resumes remembering what was said.
   */
  resume(): void {
    if (!this.state.paused || this.state.resuming) return;
    this.state.resuming = true;
    void this.#openVoice();
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
        // The engine says it would take an offer. The first one is the game
        // beginning, and the page opens its connection then.
        //
        // Every later one follows a pause: the engine lets the conversation go
        // and comes straight back round to waiting for a new offer. Acting on
        // that would resume a conversation nobody asked to resume, and start
        // billing for it again — which is the whole of what the pause was for.
        // So a second invitation is noted and left alone, and the player's own
        // press is what answers it.
        if (!this.state.invited) void this.#openVoice();
        this.state.invited = true;
        break;
      case "pause":
        // The conversation was let go — for silence, or because this player
        // asked. The session is still here; what ended is the expensive half of
        // it, which is also the answer anybody pressing pause is waiting for.
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

  /**
   * Opens the player's own audio connection, once invited.
   *
   * A failure here loses the voice rather than the session: the transcript, the
   * portrait and the character's own words all keep arriving over the stream,
   * so a player without a microphone still sees the conversation happen.
   */
  async #openVoice(): Promise<void> {
    // One negotiation at a time. A second offer while the first is still in
    // flight leaves two peers racing for one connection, and whichever loses
    // closes the one the other is waiting on an answer for.
    if (!this.voice || this.state.voice === "connecting") {
      this.state.resuming = false;
      return;
    }
    this.state.voice = "connecting";
    this.state.problem = null;
    this.#notify();
    try {
      await this.voice.connect();
      // Paused is cleared here rather than on the way in: until there is a
      // connection the conversation is still the paused one, and a control that
      // said otherwise would be saying it early.
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

  /**
   * Rebuilds everything a reloaded page missed: the wiring, where the session
   * stands, and the conversation so far. The socket carries what is happening;
   * this carries what has happened.
   */
  async restore(base: string | URL): Promise<void> {
    // The wiring is fetched first and is the reachability check: without it
    // there is no session to play, and saying so beats every symptom that
    // follows from not knowing. loadTopology says what went wrong.
    if (!(await this.loadTopology(base))) {
      this.#notify();
      return;
    }
    await this.#loadSnapshot(base);
    await this.#loadHistory(base);

    // Replay says what happened; it must not make it happen again. So the audio
    // is opened here, once, from where the session actually stands rather than
    // from an event in its past — and a conversation that was let go is left
    // alone, because picking it up is the player's to ask for.
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
    // Where the conversation stands, which a reloading page has no other way to
    // learn: the invitation it would have acted on was sent before it existed.
    // "open" counts as invited too — the conversation running there is one this
    // page has lost, and offering to take it over is what continues it.
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

  /**
   * Replayed events go through exactly the same path as live ones, so there is
   * no second way to render a conversation and no second place for it to be
   * wrong.
   */
  async #loadHistory(base: string | URL): Promise<void> {
    const response = await fetch(new URL("history", base));
    if (!response.ok) return;
    const history: unknown = await response.json();
    if (!Array.isArray(history)) return;
    for (const event of history as SessionEvent[]) {
      this.#apply(event);
    }
  }

  /**
   * Fetches the wiring once. It does not change during a session, so it is a
   * plain GET rather than something the live stream has to carry.
   */
  async loadTopology(base: string | URL): Promise<boolean> {
    let response: Response;
    try {
      response = await fetch(new URL("topology", base));
    } catch {
      // A refused or unreachable server throws rather than answering, which is
      // the only case where the server itself is the problem.
      this.state.problem = "Cannot reach the game server. Is it still running?";
      return false;
    }

    if (response.status === 404) {
      // The server answered, so it is up: what is missing is the session. Far
      // and away the likeliest cause is a link without one.
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
    // A gate that opened before the wiring arrived would otherwise leave the
    // game looking unstarted.
    this.#recomputeStarted();
    this.#notify();
    return true;
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
