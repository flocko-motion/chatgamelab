/**
 * The headless playback-engine: holds all session state with no DOM or audio device, so
 * a full game plays from node.
 */
import { detailOf, refusalOf, type Choice, type SessionRef, type Settled, type UsageReport } from "./client.js";
import { isSessionEvent, type BlockState, type Phase, type Refusal, type SessionEvent } from "./protocol.js";
import type { Conversation, ToggleName } from "./state.js";
import { emptyState, explain, type PlaybackEngineState } from "./state.js";
import { notice, type TextKey } from "./texts.js";
import type { Transport, TransportMessage } from "./transport.js";
import type { Voice } from "./voice.js";

export type Listener = (state: PlaybackEngineState) => void;

export class PlaybackEngine {
  readonly state: PlaybackEngineState = emptyState();

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
  async rebuild(session: SessionRef): Promise<void> {
    this.state.utterances = [];
    this.state.notes = [];
    this.#open = null;
    this.#nextUtterance = 1;
    this.#notify();
    await this.restore(session);
  }

  close(): void {
    this.voice?.close();
    this.transport.close();
    this.#set("closed");
  }

  /** Lets the conversation go now; `pausing` stands until the engine confirms the pause. */
  async pause(session: SessionRef): Promise<void> {
    if (this.state.pausing || this.state.paused) return;
    this.state.pausing = true;
    // Stops sending at once, while the control holds its transition until the engine confirms.
    this.voice?.setSending(false);
    this.#notify();

    try {
      await session.client.postApiEngineSessionsIdPause(session.id);
    } catch (thrown) {
      // Nothing was let go, so the conversation is live again.
      this.state.pausing = false;
      this.voice?.setSending(true);
      this.state.problem = notice("errors.pauseFailed", await detailOf(thrown));
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
  async choose(session: SessionRef, offer: string, choice?: Choice): Promise<boolean> {
    const accepted = await this.#post(
      () => session.client.postApiEngineSessionsIdChoose(session.id, choice ? { offer, choice } : { offer }),
      "errors.startFailed",
    );
    if (accepted) await this.loadOptions(session);
    return accepted;
  }

  /** Switches a feature during play; off is always allowed, on within the offer. */
  async setToggle(session: SessionRef, toggle: ToggleName, on: boolean): Promise<boolean> {
    return this.#post(() => session.client.postApiEngineSessionsIdToggles(session.id, { toggle, on }), "errors.switchFailed");
  }

  async #post(call: () => Promise<unknown>, failure: TextKey): Promise<boolean> {
    try {
      await call();
      this.state.problem = null;
      this.#notify();
      return true;
    } catch (thrown) {
      this.state.problem = notice(failure, await detailOf(thrown));
      this.#notify();
      return false;
    }
  }

  /** Fetches what the player may choose from, and what the session settled on. */
  async loadOptions(session: SessionRef): Promise<void> {
    let options;
    try {
      ({ data: options } = await session.client.getApiEngineSessionsIdOptions(session.id));
    } catch {
      return;
    }
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
  until(predicate: (state: PlaybackEngineState) => boolean, timeoutMs = 10_000): Promise<PlaybackEngineState> {
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
        // Something answered, so a fault reported earlier has passed.
        this.state.fault = null;
        break;
      case "problem":
        this.#applyProblem(event.Value);
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
      this.state.problem = notice("errors.voiceFailed", error);
    }
    this.state.resuming = false;
    this.#notify();
  }

  /** Rebuilds what a reloaded page missed; the socket carries only what is happening. */
  async restore(session: SessionRef): Promise<void> {
    // The wiring is the reachability check; loadTopology says what went wrong.
    if (!(await this.loadTopology(session))) {
      this.#notify();
      return;
    }
    await this.#loadSnapshot(session);
    await this.loadOptions(session);
    await this.#loadHistory(session);

    // Audio opens here from where the session stands, so replay never resumes a pause.
    if (this.state.invited && !this.state.paused && this.state.voice === "idle") {
      void this.#openVoice();
    }
  }

  async #loadSnapshot(session: SessionRef): Promise<void> {
    let snapshot;
    try {
      ({ data: snapshot } = await session.client.getApiEngineSessionsIdState(session.id));
    } catch {
      return;
    }
    this.state.phases = snapshot.phases as Record<string, Phase>;
    this.state.props = snapshot.props;
    this.state.usage = snapshot.usage;
    this.state.started = snapshot.started;
    this.state.choosing = snapshot.choosing;
    this.state.settled = snapshot.settled ?? null;
    // The invitation predates a reloaded page, so it asks where the conversation stands.
    // "open" is a conversation this page lost, so it offers again.
    switch (snapshot.conversation as Conversation | undefined) {
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
  async #loadHistory(session: SessionRef): Promise<void> {
    let history: unknown;
    try {
      ({ data: history } = await session.client.getApiEngineSessionsIdHistory(session.id));
    } catch {
      return;
    }
    if (!Array.isArray(history)) return;
    for (const event of history) {
      if (isSessionEvent(event)) this.#apply(event);
    }
  }

  /** Fetches the wiring once, as it never changes during a session. */
  async loadTopology(session: SessionRef): Promise<boolean> {
    let topology;
    try {
      ({ data: topology } = await session.client.getApiEngineSessionsIdTopology(session.id));
    } catch (thrown) {
      // The engine refuses with a reason: unknown, or stored too long ago to resume.
      const refusal = await refusalOf(thrown);
      if (refusal) this.refuse(refusal);
      // Only an unreachable server throws something other than its answer.
      else this.state.problem = notice("errors.serverUnreachable");
      return false;
    }

    this.state.problem = null;
    this.state.refusal = null;
    this.state.topology = topology;
    // A gate that opened before the wiring arrived would otherwise look unstarted.
    this.#recomputeStarted();
    this.#notify();
    return true;
  }

  /** Shows why the session cannot be had, such as a launch the platform refused. */
  refuse(refusal: Refusal): void {
    this.state.refusal = refusal;
    this.state.problem = explain(refusal);
    this.#notify();
  }

  /** A key failing during play: told apart from a lost connection, and not blocking. */
  #applyProblem(value: string): void {
    try {
      const refusal = JSON.parse(value) as Refusal;
      this.state.fault = explain(refusal);
    } catch {
      // A malformed report costs the notice, not the game.
    }
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

  #set(connection: PlaybackEngineState["connection"]): void {
    this.state.connection = connection;
    this.#notify();
  }

  #notify(): void {
    for (const listener of this.#listeners) listener(this.state);
  }
}

type Utterance = PlaybackEngineState["utterances"][number];

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
