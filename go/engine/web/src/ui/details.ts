/**
 * Two of the tabs under the graph: the selected block or wire, and the whole
 * session's usage. Reading a stage costs a click rather than a dialog, and the
 * graph stays visible while you read; the usage stays current whatever is
 * selected.
 */
import type {
  BlockPrompt,
  EdgeDetail,
  NodeDetail,
  ObserverState,
  Sample,
  SessionRef,
  ThreadFill,
  TranscriptLine,
  UsageRecord,
} from "../client.js";
import type { PlaybackEngineState } from "../state.js";
import { blockLabel, promptLabel, settingLabel, text as said, type TextKey } from "../texts.js";

/** How a live model takes an injection; the engine lists the kinds a block takes. */
type InjectionKind = "direction" | "background" | "speech";

const INJECTION_TEXTS: Record<InjectionKind, { label: TextKey; placeholder: TextKey }> = {
  direction: { label: "pipeline.injectDirection", placeholder: "pipeline.injectDirectionHint" },
  background: { label: "pipeline.injectBackground", placeholder: "pipeline.injectBackgroundHint" },
  speech: { label: "pipeline.injectSpeech", placeholder: "pipeline.injectSpeechHint" },
};

function isInjectionKind(kind: string): kind is InjectionKind {
  return kind in INJECTION_TEXTS;
}

export class Details {
  #selected: string | null = null;
  /**
   * The block whose thread is compacting at the reader's request, and whether
   * it has reported working since; its details are fetched again once it is
   * ready, so the new fill shows.
   */
  #compacting: { name: string; worked: boolean } | null = null;
  /** The kind of injection last chosen, offered again in the next box. */
  #injectionKind: InjectionKind = "direction";
  /** Fetches a selected observer's state again every second, while it is selected. */
  #observerPoll: ReturnType<typeof setInterval> | null = null;
  /** Whether this is a dev server, which may tune an observer's beat. */
  #dev = false;

  constructor(
    private readonly title: HTMLElement,
    private readonly body: HTMLElement,
    private readonly usage: HTMLElement,
    private readonly session: SessionRef,
  ) {
    this.#renderNothingSelected();
  }

  /** Lets go of the selected block or wire. */
  clear(state: PlaybackEngineState): void {
    this.#stopObserverPoll();
    this.#selected = null;
    this.#renderNothingSelected();
    this.#renderSession(state);
  }

  #renderNothingSelected(): void {
    this.title.textContent = "";
    this.body.replaceChildren(text("p", "meta", said("insights.pickBlock")));
  }

  /** Shows what one wire has carried. */
  async showEdge(from: string, to: string, kind: string): Promise<void> {
    this.#stopObserverPoll();
    this.#selected = `${from}→${to}`;
    this.title.textContent = `${from} → ${to}`;
    this.body.replaceChildren(text("p", "meta", "loading…"));

    try {
      let detail: EdgeDetail;
      try {
        ({ data: detail } = await this.session.client.getApiEngineSessionsIdEdgesFromToKind(
          this.session.id,
          encodeURIComponent(from),
          encodeURIComponent(to),
          encodeURIComponent(kind),
        ));
      } catch (thrown) {
        if (!(thrown instanceof Response)) throw thrown;
        this.body.replaceChildren(text("p", "meta", "no detail for this wire"));
        return;
      }

      const parts: HTMLElement[] = [];
      const facts = document.createElement("div");
      facts.className = "kv";
      facts.append(text("span", "k", "carries"), text("span", "v", detail.kind));
      parts.push(facts);

      if (detail.recent.length) parts.push(samples("last carried", detail.recent));
      else parts.push(text("p", "meta", "nothing has crossed this wire yet"));

      this.body.replaceChildren(...parts);
    } catch (error) {
      this.body.replaceChildren(text("p", "meta", `could not read this wire: ${String(error)}`));
    }
  }

  /** Shows one block, fetched fresh so its recent values are current. */
  async show(name: string): Promise<void> {
    this.#stopObserverPoll();
    this.#selected = name;
    this.title.textContent = name;
    this.body.replaceChildren(text("p", "meta", "loading…"));

    try {
      await this.#renderNode(name);
    } catch (error) {
      this.body.replaceChildren(text("p", "meta", `could not read ${name}: ${String(error)}`));
    }
  }

  /**
   * Repaints the usage on a state change. A selected block is left alone:
   * refetching it on every event would flicker, and its figures are a
   * moment's snapshot rather than a live reading.
   */
  refresh(state: PlaybackEngineState): void {
    this.#dev = !!state.topology?.dev;
    const compacting = this.#compacting;
    if (compacting && compacting.name === this.#selected) {
      const phase = state.phases[compacting.name];
      if (phase === "working") compacting.worked = true;
      if (compacting.worked && phase === "ready") {
        this.#compacting = null;
        void this.show(compacting.name);
      }
    }
    this.#renderSession(state);
  }

  #renderSession(state: PlaybackEngineState): void {
    const usage = state.usage;
    if (!usage) {
      this.usage.replaceChildren(text("p", "meta", "nothing spent yet"));
      return;
    }

    // Go encodes an empty slice as null, which a session that has spent nothing
    // yet reports.
    const rows = (usage.byModel ?? []).flatMap((record) => [
      text("span", "k", record.model),
      text("span", "v", `${amount(record)} · ${money(record.cost)}`),
    ]);

    const label = text("span", "k total", usage.complete ? "total" : "total (partly unpriced)");
    const total = text("span", "v total", usage.complete ? money(usage.totalCost) : `${money(usage.totalCost)}+`);

    const list = document.createElement("div");
    list.className = "kv";
    list.append(...rows, label, total);

    this.usage.replaceChildren(list);
  }

  #stopObserverPoll(): void {
    if (this.#observerPoll) clearInterval(this.#observerPoll);
    this.#observerPoll = null;
  }

  async #pollObserver(name: string, pane: HTMLElement): Promise<void> {
    if (this.#selected !== name || !pane.isConnected) {
      this.#stopObserverPoll();
      return;
    }
    try {
      const { data } = await this.session.client.getApiEngineSessionsIdNodesNode(this.session.id, encodeURIComponent(name));
      if (data.observer && this.#selected === name) drawObserver(pane, data.observer);
    } catch {
      // A missed reading is made up by the next.
    }
  }

  /** A dev server's knob for how long fragments wait before the model sees them. */
  #beatControl(name: string, beatMs: number): HTMLElement {
    const form = document.createElement("form");
    form.className = "observer-beat";
    const field = document.createElement("input");
    field.type = "number";
    field.min = "100";
    field.max = "10000";
    field.step = "100";
    field.value = String(beatMs);
    const button = document.createElement("button");
    button.type = "submit";
    button.textContent = said("director.setBeat");
    const status = text("span", "meta", "");
    form.append(field, button, status);
    form.addEventListener("submit", (event) => {
      event.preventDefault();
      void this.session.client
        .postApiEngineSessionsIdNodesNodeBeat(this.session.id, encodeURIComponent(name), { ms: Number(field.value) })
        .then(() => (status.textContent = said("director.beatSet")))
        .catch(() => (status.textContent = said("director.beatRefused")));
    });
    return form;
  }

  async #renderNode(name: string): Promise<void> {
    let detail: NodeDetail;
    try {
      ({ data: detail } = await this.session.client.getApiEngineSessionsIdNodesNode(this.session.id, encodeURIComponent(name)));
    } catch (thrown) {
      if (!(thrown instanceof Response)) throw thrown;
      this.body.replaceChildren(text("p", "meta", `no detail for ${name}`));
      return;
    }
    // A game setting is named by its field and the block that reads it, and
    // shows only its value: it is no block, so it has no type or state.
    if (detail.role === "setting") {
      const [field = "", reader = ""] = name.split(">");
      this.title.textContent = reader
        ? `${settingLabel(field, name)} → ${blockLabel(reader)}`
        : blockLabel(name);
      const value = detail.outputs?.[0]?.value ?? "";
      this.body.replaceChildren(text("p", "sample-value", value || said("insights.settingEmpty")));
      return;
    }
    const parts: HTMLElement[] = [facts(detail)];
    // The prompts the block runs on, each open to a change: a player tries
    // how the answers change, from the block's next call on.
    if (detail.prompts?.length) parts.push(this.#promptsSection(detail.prompts));
    // A block that takes injections, such as an observer, takes them here,
    // so a player can try how the conversation it watches reacts.
    if (detail.injectable) parts.push(this.#injectionBox(name, detail.injections ?? []));

    // A block following a conversation from the side shows where it stands,
    // kept current while it is selected: its state changes every second.
    if (detail.observer) {
      const pane = document.createElement("div");
      pane.className = "observer-pane";
      drawObserver(pane, detail.observer);
      parts.push(pane);
      if (this.#dev) parts.push(this.#beatControl(name, detail.observer.beatMs));
      this.#observerPoll = setInterval(() => void this.#pollObserver(name, pane), 1000);
    }

    // A block holding a conversation shows it whole: its outputs are a window
    // of the last few pieces, which audio alone fills.
    if (detail.transcript?.length) parts.push(transcript(detail.transcript));

    // A sink has no outputs and a source has no inputs, so neither is assumed.
    if (detail.outputs?.length) parts.push(samples("last outputs", detail.outputs));
    if (detail.inputs?.length) parts.push(samples("last inputs", detail.inputs));
    // A block on a thread can have it compacted now, to see how well a
    // summary keeps the story.
    if (detail.thread) parts.push(this.#compactButton(name));

    this.body.replaceChildren(...parts);
  }

  #promptsSection(prompts: BlockPrompt[]): HTMLElement {
    const section = document.createElement("section");
    section.className = "prompts";
    section.append(text("h4", "", said("prompts.title")), text("p", "meta", said("prompts.hint")));
    for (const prompt of prompts) section.append(this.#promptEditor(prompt));
    return section;
  }

  /** One prompt, folded under its name, with its text to change and a way back. */
  #promptEditor(prompt: BlockPrompt): HTMLElement {
    const fold = document.createElement("details");
    const summary = document.createElement("summary");
    summary.textContent = promptLabel(prompt.name);
    const mark = text("span", "edited-mark", said("prompts.edited"));
    mark.hidden = !prompt.edited;
    summary.append(" ", mark);

    const form = document.createElement("form");
    form.className = "injection";
    const box = document.createElement("textarea");
    box.rows = 8;
    box.value = prompt.text;
    box.setAttribute("aria-label", promptLabel(prompt.name));
    const save = document.createElement("button");
    save.type = "submit";
    save.className = "injection-send";
    save.textContent = said("prompts.save");
    const reset = document.createElement("button");
    reset.type = "button";
    reset.className = "injection-send";
    reset.textContent = said("prompts.reset");
    reset.hidden = !prompt.edited;
    reset.addEventListener("click", () => {
      box.value = prompt.default;
      form.requestSubmit();
    });
    const actions = document.createElement("div");
    actions.className = "prompt-actions";
    actions.append(save, reset);
    const status = text("p", "meta", "");
    form.append(box, actions, status);

    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      save.disabled = true;
      status.textContent = said("prompts.saving");
      try {
        await this.session.client.putApiEngineSessionsIdPromptsName(
          this.session.id,
          encodeURIComponent(prompt.name),
          { text: box.value },
        );
        const edited = box.value !== prompt.default;
        mark.hidden = !edited;
        reset.hidden = !edited;
        status.textContent = said("prompts.saved");
      } catch {
        status.textContent = said("prompts.failed");
      } finally {
        save.disabled = false;
      }
    });

    fold.append(summary, form);
    return fold;
  }

  #injectionBox(name: string, kinds: string[]): HTMLElement {
    const form = document.createElement("form");
    form.className = "injection";

    const title = text("span", "injection-title", said("pipeline.injection"));

    // The kind decides how the model takes the text, so it is chosen first and
    // kept for the next one: trying the same kind again is the common case.
    const known = kinds.filter(isInjectionKind);
    if (!known.includes(this.#injectionKind)) this.#injectionKind = known[0] ?? "direction";
    const switcher = document.createElement("div");
    switcher.className = "injection-kinds";
    switcher.setAttribute("role", "radiogroup");
    switcher.setAttribute("aria-label", said("pipeline.injection"));

    const box = document.createElement("textarea");
    box.rows = 3;
    box.setAttribute("aria-label", said("pipeline.injection"));

    const choose = (kind: InjectionKind): void => {
      this.#injectionKind = kind;
      for (const option of switcher.querySelectorAll<HTMLButtonElement>("button")) {
        option.setAttribute("aria-checked", String(option.value === kind));
      }
      box.placeholder = said(INJECTION_TEXTS[kind].placeholder);
    };
    for (const kind of known) {
      const option = document.createElement("button");
      option.type = "button";
      option.value = kind;
      option.setAttribute("role", "radio");
      option.textContent = said(INJECTION_TEXTS[kind].label);
      option.addEventListener("click", () => choose(kind));
      switcher.append(option);
    }
    choose(this.#injectionKind);

    const send = document.createElement("button");
    send.type = "submit";
    send.className = "injection-send";
    send.textContent = said("pipeline.injectionSend");
    const status = text("p", "meta", "");

    form.append(title, switcher, box, send, status);

    // Enter sends, Shift+Enter starts a new line.
    box.addEventListener("keydown", (event) => {
      if (event.key === "Enter" && !event.shiftKey && !event.isComposing) {
        event.preventDefault();
        form.requestSubmit();
      }
    });

    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      const injection = box.value.trim();
      if (!injection) return;
      send.disabled = true;
      status.textContent = said("pipeline.injectionSending");
      try {
        await this.session.client.postApiEngineSessionsIdNodesNodeInject(
          this.session.id,
          encodeURIComponent(name),
          { kind: this.#injectionKind, text: injection },
        );
        box.value = "";
        status.textContent = said("pipeline.injectionSent");
      } catch {
        status.textContent = said("pipeline.injectionFailed");
      } finally {
        send.disabled = false;
        box.focus();
      }
    });
    return form;
  }

  #compactButton(name: string): HTMLElement {
    const button = document.createElement("button");
    button.type = "button";
    button.className = "compact";
    button.textContent = said("pipeline.compact");
    button.addEventListener("click", async () => {
      button.disabled = true;
      button.textContent = said("pipeline.compacting");
      try {
        await this.session.client.postApiEngineSessionsIdNodesNodeCompact(this.session.id, encodeURIComponent(name));
        this.#compacting = { name, worked: false };
      } catch {
        button.disabled = false;
        button.textContent = said("pipeline.compactFailed");
      }
    });
    return button;
  }
}

function facts(detail: NodeDetail): HTMLElement {
  const rows: [string, string][] = [
    ["type", detail.type],
    ["role", detail.role],
    ["state", detail.phase],
  ];
  if (detail.usage?.model) rows.push(["model", detail.usage.model]);
  if (detail.thread) rows.push(["context", fill(detail.thread)]);
  if (detail.usage) rows.push(["spent", `${amount(detail.usage)} · ${money(detail.usage.cost)}`]);

  const list = document.createElement("div");
  list.className = "kv";
  for (const [key, value] of rows) {
    list.append(text("span", "k", key), text("span", "v", value));
  }
  return list;
}

/** Seconds, to a tenth, from milliseconds, in the reader's own way of writing numbers. */
function seconds(ms: number): string {
  return `${(ms / 1000).toLocaleString(undefined, { minimumFractionDigits: 1, maximumFractionDigits: 1 })} s`;
}

/** A text with its {{placeholders}} filled in. */
function filled(key: TextKey, values: Record<string, string | number>): string {
  return said(key).replace(/\{\{(\w+)\}\}/g, (_, name: string) => String(values[name] ?? ""));
}

/** How long ago something happened, or that it has not yet. */
function ago(ms: number): string {
  return ms < 0 ? said("director.notYet") : filled("director.ago", { s: seconds(ms) });
}

const ENTRY_TEXTS: Record<string, TextKey> = {
  line: "director.entryLine",
  silence: "director.entrySilence",
  injection: "director.entryInjection",
  thought: "director.entryThought",
  direction: "director.entryDirection",
  background: "director.entryBackground",
  dropped: "director.entryDropped",
  error: "director.entryError",
  conversation: "director.entryConversation",
  rules: "director.entryRules",
  absent: "director.entryAbsent",
};

/** Who an entry names: a speaker, or the kind of a hand-made injection. */
function entrySpeaker(speaker: string): string {
  if (speaker === "character") return said("director.character");
  if (speaker === "player") return said("director.player");
  return isInjectionKind(speaker) ? said(INJECTION_TEXTS[speaker].label) : speaker;
}

/** Rows of a key and its value, as the details lay them out. */
function table(rows: [string, string][]): HTMLElement {
  const list = document.createElement("div");
  list.className = "kv";
  for (const [k, v] of rows) list.append(text("span", "k", k), text("span", "v", v));
  return list;
}

/**
 * Where an observer stands: what it is doing, how long each side has been
 * quiet, what it has counted, and its log, newest at the bottom. Drawn anew
 * on every reading, the log kept at its end when the reader was there.
 */
function drawObserver(pane: HTMLElement, state: ObserverState): void {
  const oldLog = pane.querySelector<HTMLElement>(".observer-log");
  const atEnd = !oldLog || oldLog.scrollTop + oldLog.clientHeight >= oldLog.scrollHeight - 4;
  const keptScroll = oldLog?.scrollTop ?? 0;
  const f = state.figures;

  const now = document.createElement("section");
  now.append(
    text("h4", "", said("director.now")),
    table([
      [
        said("director.doing"),
        state.thinking
          ? filled("director.thinking", { s: seconds(state.thinkingForMs) })
          : state.standingDown
            ? filled("director.standingDown", { s: seconds(state.absentMs) })
            : said("director.listening"),
      ],
      [said("director.characterSpoke"), ago(state.characterAgoMs)],
      [said("director.playerSpoke"), ago(state.playerAgoMs)],
      [said("director.quietFor"), state.quietForMs < 0 ? "—" : seconds(state.quietForMs)],
      [said("director.pool"), filled("director.poolValue", { n: state.pool, ms: state.beatMs })],
      [said("director.gap"), seconds(state.gapMs)],
      [said("director.waiting"), filled("director.lines", { n: state.pending })],
      [said("director.inARow"), filled("director.ofLimit", { n: state.inARow, limit: state.inARowLimit })],
    ]),
  );

  const calls = filled("director.callsValue", { n: f.calls, s: seconds(f.lastCallMs) });
  const counted = document.createElement("section");
  counted.append(
    text("h4", "", said("director.counted")),
    table([
      [said("director.fragments"), String(f.fragments)],
      [said("director.beats"), filled("director.beatsValue", { n: f.flushes, lines: f.lines })],
      [said("director.late"), f.late ? filled("director.lateValue", { n: f.late, ms: f.largestLateMs }) : "0"],
      [said("director.notices"), String(f.silenceNotices)],
      [said("director.quietReplies"), String(f.quiet)],
      [said("director.calls"), f.errors ? `${calls} · ${filled("director.failed", { n: f.errors })}` : calls],
      [said("director.sent"), filled("director.sentValue", { d: f.directions, b: f.backgrounds })],
      [said("director.reminders"), String(f.reminders)],
      [said("director.dropped"), filled("director.droppedValue", { n: f.dropped })],
      [said("director.injected"), String(f.injected)],
    ]),
  );

  const logSection = document.createElement("section");
  logSection.append(text("h4", "", said("director.log")));
  const log = document.createElement("div");
  log.className = "observer-log";
  for (const entry of state.log ?? []) {
    const row = document.createElement("div");
    row.className = `observer-entry ${entry.kind}`;
    const head = [clock(entry.at), ENTRY_TEXTS[entry.kind] ? said(ENTRY_TEXTS[entry.kind]!) : entry.kind];
    if (entry.speaker) head.push(entrySpeaker(entry.speaker));
    if (entry.late) head.push(said("director.lateMark"));
    row.append(text("span", "sample-kind", head.join(" · ")), text("span", "sample-value", entry.text));
    log.append(row);
  }
  if (!state.log?.length) log.append(text("p", "meta", said("director.empty")));
  logSection.append(log);

  pane.replaceChildren(now, counted, logSection);
  log.scrollTop = atEnd ? log.scrollHeight : keptScroll;
}

/** The conversation so far, oldest first, read like a chat: newest at the bottom. */
function transcript(lines: TranscriptLine[]): HTMLElement {
  const section = document.createElement("section");
  section.append(text("h4", "", "transcript"));
  const list = document.createElement("div");
  list.className = "transcript";
  for (const line of lines) {
    const row = document.createElement("div");
    row.className = `transcript-line ${line.speaker}`;
    row.append(text("span", "sample-kind", line.speaker), text("span", "sample-value", line.text));
    list.append(row);
  }
  section.append(list);
  // Opened at its end, where a conversation that stopped is read from.
  requestAnimationFrame(() => {
    list.scrollTop = list.scrollHeight;
  });
  return section;
}

function samples(title: string, entries: Sample[]): HTMLElement {
  const section = document.createElement("section");
  section.append(text("h4", "", `${title} — newest first`));
  // Newest first, because that is what a reader came for; the time settles any
  // remaining question about order.
  for (const entry of [...entries].reverse()) {
    const row = document.createElement("div");
    row.className = "sample";
    row.append(
      // An edge sample needs no peer: both ends are already in the heading.
      text(
        "span",
        "sample-kind",
        [clock(entry.at), entry.kind, entry.peer].filter(Boolean).join(" · "),
      ),
      // An empty value is the marker closing an utterance, which is worth
      // naming rather than rendering as a blank line.
      text("span", "sample-value", entry.value || "(end of utterance)"),
    );
    section.append(row);
  }
  return section;
}

/** Time of day to the millisecond: enough to order events that arrive together. */
function clock(at: string): string {
  const when = new Date(at);
  if (Number.isNaN(when.getTime())) return "";
  return when.toLocaleTimeString(undefined, { hour12: false }) +
    "." + String(when.getMilliseconds()).padStart(3, "0");
}

/**
 * How full a thread is, as a share of its model's window: "42% · 2,520 of
 * 6,000 tok". Nothing before a call, or after a compaction until the next.
 */
function fill(thread: ThreadFill): string {
  if (!thread.tokens) return "—";
  const tokens = thread.tokens.toLocaleString();
  if (!thread.window) return `${tokens} tok`;
  const share = Math.round((thread.tokens / thread.window) * 100);
  return `${share}% · ${tokens} of ${thread.window.toLocaleString()} tok`;
}

function amount(record: UsageRecord): string {
  const parts: string[] = [];
  if (record.images) parts.push(`${record.images} img`);
  if (record.audioSeconds) parts.push(`${record.audioSeconds.toFixed(0)}s`);
  const tokens = (record.inputTokens ?? 0) + (record.cachedInputTokens ?? 0) + (record.outputTokens ?? 0);
  if (tokens) parts.push(`${tokens} tok`);
  // A speaking model's audio is billed apart from its text, so it is shown apart.
  const audio = (record.audioInputTokens ?? 0) + (record.cachedAudioInputTokens ?? 0) + (record.audioOutputTokens ?? 0);
  if (audio) parts.push(`${audio} audio tok`);
  return parts.join(" ") || "—";
}

/** Cost is an estimate from a hand-maintained table, so it is shown coarsely. */
function money(cost: number): string {
  if (!cost) return "—";
  if (cost < 0.01) return "<$0.01";
  return `$${cost.toFixed(2)}`;
}

function text(tag: string, className: string, content: string): HTMLElement {
  const element = document.createElement(tag);
  if (className) element.className = className;
  element.textContent = content;
  return element;
}
