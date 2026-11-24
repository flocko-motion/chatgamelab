/**
 * The headless playback-engine: holds all session state with no DOM or audio device, so
 * a full game plays from node.
 */
import {
  detailOf,
  refusalOf,
  type Choice,
  type JournalEntry,
  type SessionRef,
  type Settled,
  type UsageReport,
} from "./client.js";

/** The reason on an invitation that waits for the player's press. */
const INVITE_PRESS = "press";

/** How many journal entries the page keeps, as many as the engine does. */
const JOURNAL_KEPT = 2000;
import {
  AUDIO_DISCARD_BYTES,
  ERROR_CODES,
  isSessionEvent,
  TEXT_DISCARD,
  TEXT_REPLACE,
  type BlockState,
  type ErrorReport,
  type Phase,
  type Props,
  type Refusal,
  type SessionEvent,
} from "./protocol.js";
import type { Conversation, ToggleName } from "./state.js";
import { emptyState, explain, overReason, type PlaybackEngineState } from "./state.js";
import { notice, type Notice, type TextKey } from "./texts.js";
import type { Transport, TransportMessage } from "./transport.js";
import type { Speaker } from "./speaker.js";
import type { Voice } from "./voice.js";

export type Listener = (state: PlaybackEngineState) => void;

/** How many runs the Insights feed keeps; older ones fall off its top. */
const MAX_RUNS = 200;

export class PlaybackEngine {
  readonly state: PlaybackEngineState = emptyState();

  #listeners = new Set<Listener>();
  #nextUtterance = 1;
  /** The last run's id. */
  #runs = 0;
  #open: Utterance | null = null;
  /** The player's last line, typed or as heard, given back if the engine drops its turn. */
  #said: string | null = null;
  /** Typed lines sent and not yet back on `said`, oldest first. */
  #typed: string[] = [];
  /** Spoken clips sent whose words have not come back on `said`, nor failed. */
  #clips = 0;

  /**
   * The voice and the speaker are optional, so a headless run can watch a
   * conversation it can neither join nor hear.
   */
  constructor(
    private readonly transport: Transport,
    private readonly voice?: Voice,
    private readonly speaker?: Speaker,
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
    // What was heard of a clip still pending comes back with the history.
    this.#typed = [];
    this.#clips = 0;
    this.state.hearing = false;
    this.#notify();
    await this.restore(session);
  }

  close(): void {
    this.voice?.close();
    this.speaker?.discard();
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

  /**
   * Sends one clip of the player's speech, PCM16 at 24 kHz mono, as one
   * utterance. The player's line is pending until the engine says what it
   * heard, on `said`, or that nothing came through.
   */
  speak(clip: ArrayBuffer): void {
    this.#clips++;
    this.state.hearing = true;
    this.state.redo = null;
    this.state.fault = null;
    this.transport.send(clip);
    this.#notify();
  }

  /** Sends typed input, which live and turn-based genres both take. */
  say(text: string): void {
    this.#said = text;
    this.#typed.push(text);
    this.state.redo = null;
    this.transport.send(JSON.stringify({ text }));
  }

  /** Tells the player of a fault that leaves the controls working, such as a microphone refused. */
  fault(said: Notice): void {
    this.state.fault = said;
    this.#notify();
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
      this.#hear(data);
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
        // An empty delta closes the utterance, ordered against the text it ends;
        // a discard drops it unfinished, as a broken reply is made again.
        if (event.Value === "") this.takeFloor();
        else if (event.Value === TEXT_DISCARD) this.#discard();
        else if (event.Value.startsWith(TEXT_REPLACE)) this.#replace(event.Value.slice(TEXT_REPLACE.length));
        else this.#extend(event.Value);
        // Something answered, so a fault reported earlier has passed.
        this.state.fault = null;
        break;
      case "said":
        this.#applySaid(event.Value);
        break;
      case "problem":
        this.#applyProblem(event.Value);
        break;
      case "image":
        this.state.image = event.Value;
        break;
      case "props":
        this.state.props = parseProps(event.Value) ?? this.state.props;
        break;
      case "error":
        this.#applyError(event.Value);
        break;
      case "state":
        this.#applyBlockState(event.Value);
        break;
      case "usage":
        this.#applyUsage(event.Value);
        break;
      case "connect":
        // An invitation that waits for the press follows a pause, or a conversation that could
        // not go on by itself: answering it would re-bill one nobody asked to resume. Any other
        // is answered at once, the first and one after the provider dropped the conversation.
        if (event.Value === INVITE_PRESS) {
          this.state.paused = true;
        } else {
          void this.#openVoice();
        }
        this.state.invited = true;
        break;
      case "settled":
        this.#applySettled(event.Value);
        break;
      case "journal":
        this.#applyJournal(event.Value);
        break;
      case "heartbeat":
        // Only proof the socket lives, which the transport has already taken.
        return;
      case "over":
        // Final: the conversation is let go for good, and nothing offers to resume it.
        this.state.over = overReason(event.Value);
        this.state.paused = false;
        this.state.pausing = false;
        this.state.resuming = false;
        this.state.invited = false;
        this.state.voice = "idle";
        this.voice?.close();
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
    // A conversation the provider dropped leaves its audio connection behind; a new one replaces it.
    if (this.state.voice === "live") this.voice.close();
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
    await this.#loadJournal(session);

    // Audio opens here from where the session stands, so replay never resumes a pause.
    if (this.state.invited && !this.state.paused && !this.state.over && this.state.voice === "idle") {
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
    if (snapshot.over) this.state.over = overReason(snapshot.over.reason);
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

  /** The journal so far replaces what the page held; later entries arrive as events. */
  async #loadJournal(session: SessionRef): Promise<void> {
    let journal: unknown;
    try {
      ({ data: journal } = await session.client.getApiEngineSessionsIdJournal(session.id));
    } catch {
      return;
    }
    if (!Array.isArray(journal)) return;
    this.state.journal = (journal as JournalEntry[]).slice(-JOURNAL_KEPT);
    this.#notify();
  }

  #applyJournal(value: string): void {
    let entry: JournalEntry;
    try {
      entry = JSON.parse(value) as JournalEntry;
    } catch {
      return;
    }
    this.state.journal.push(entry);
    if (this.state.journal.length > JOURNAL_KEPT) this.state.journal.splice(0, this.state.journal.length - JOURNAL_KEPT);
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

  /**
   * An error the game goes on after, reported as its code and the text the
   * failed block was given. A dropped turn of a line this page sent is said to
   * the player, and the line is given back to send again with one press; one
   * replayed from history is long past. A turn that failed before any prose
   * is the opening's, said so the player knows to begin; the next text clears
   * it. A picture that failed is said, and the turn stands. A clip of which
   * nothing came through ends its wait, and the player is told to speak again
   * or type. Anything else is kept as a note.
   */
  #applyError(value: string): void {
    const report = parseErrorReport(value);
    if (report === null) {
      this.state.notes.push(`error: ${value}`);
      return;
    }
    switch (report.code) {
      case "turn_failed":
        if (this.#said === null) {
          if (!this.state.utterances.some((u) => u.speaker === "npc" && u.complete && u.text !== "")) {
            this.state.fault = notice("errors.openingFailed");
          }
          break;
        }
        this.state.fault = notice("errors.turnFailed");
        this.state.redo = this.#said;
        this.#said = null;
        break;
      case "image_failed":
        this.state.fault = notice("errors.imageFailed");
        break;
      case "voice_failed":
        // Only a clip this page sent: one replayed from history is long past.
        if (this.#clips === 0) break;
        this.#settleClip();
        this.state.fault = notice("errors.notHeard");
        break;
    }
  }

  /** Ends the wait of the oldest clip sent, heard or failed. */
  #settleClip(): void {
    this.#clips--;
    this.state.hearing = this.#clips > 0;
  }

  /**
   * A line of the player's, typed or as the engine heard it: it ends the
   * reply before it and joins the conversation. A line heard from a clip this
   * page sent ends that clip's wait, and is the line given back if its turn
   * is dropped; a typed one was kept when it was sent.
   */
  #applySaid(line: string): void {
    this.takeFloor();
    this.state.utterances.push({ id: this.#nextUtterance++, speaker: "player", text: line, complete: true });
    if (this.#typed[0] === line) {
      this.#typed.shift();
    } else if (this.#clips > 0) {
      this.#settleClip();
      this.#said = line;
    }
    // Something answered, so a fault reported earlier has passed.
    this.state.fault = null;
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

    const was = this.state.phases[report.node];
    this.state.phases[report.node] = report.phase;
    this.#recordRun(report.node, was, report.phase);
    this.#recomputeStarted();
  }

  /** A block that starts working is a new run; one that is ready again finished its latest. */
  #recordRun(node: string, was: Phase | undefined, phase: Phase): void {
    const runs = this.state.runs;
    if (phase === "working" && was !== "working") {
      runs.push({ id: ++this.#runs, node, done: false });
      if (runs.length > MAX_RUNS) runs.splice(0, runs.length - MAX_RUNS);
      return;
    }
    if (phase === "ready") {
      for (let i = runs.length - 1; i >= 0; i--) {
        const run = runs[i]!;
        if (run.node === node && !run.done) {
          run.done = true;
          break;
        }
      }
    }
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

  /**
   * Plays a frame of spoken reply as it arrives, queued after the one before;
   * a discard stops the speech and drops the queue, so a reply made again
   * speaks from its start. A live voice connection plays the character
   * itself, so the frames that mirror it are only counted.
   */
  #hear(frame: ArrayBuffer): void {
    if (frame.byteLength === AUDIO_DISCARD_BYTES) {
      this.state.audioDiscards++;
      this.speaker?.discard();
      return;
    }
    this.state.audioChunks++;
    if (this.state.voice !== "live") this.speaker?.play(frame);
  }

  #extend(delta: string): void {
    if (!this.#open) {
      this.#open = { id: this.#nextUtterance++, speaker: "npc", text: "", complete: false };
      this.state.utterances.push(this.#open);
    }
    this.#open.text += delta;
  }

  /** Replaces the text of the utterance still open, as when a reply was tidied after it streamed. */
  #replace(text: string): void {
    if (this.#open) this.#open.text = text;
    else this.#extend(text);
  }

  /**
   * Drops the text of the utterance still open, which is never finished. It
   * stays open, empty, so the retried reply fills the same place: the feed is
   * a timeline, and nothing is taken out of it.
   */
  #discard(): void {
    if (this.#open) this.#open.text = "";
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

/** Reads an error event's value, a report of a known code; anything else is null. */
function parseErrorReport(value: string): ErrorReport | null {
  let parsed: unknown;
  try {
    parsed = JSON.parse(value);
  } catch {
    return null;
  }
  if (typeof parsed !== "object" || parsed === null) return null;
  const report = parsed as Partial<ErrorReport>;
  if (typeof report.code !== "string" || !(ERROR_CODES as readonly string[]).includes(report.code)) return null;
  return { code: report.code, text: typeof report.text === "string" ? report.text : "" };
}

/** Reads a props event's value, a JSON object of strings; one that does not read is null. */
function parseProps(value: string): Props | null {
  let parsed: unknown;
  try {
    parsed = JSON.parse(value);
  } catch {
    return null;
  }
  if (typeof parsed !== "object" || parsed === null || Array.isArray(parsed)) return null;
  const props: Record<string, string> = {};
  for (const [name, field] of Object.entries(parsed)) {
    if (typeof field === "string") props[name] = field;
  }
  return props;
}
