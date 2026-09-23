/**
 * The details panel under the graph.
 *
 * It shows the whole session's figures by default and one block's when a block
 * is selected, in the same place — so reading a stage costs a click rather than
 * a dialog, and the graph stays visible while you read.
 */
import type { UsageRecord } from "../protocol.js";
import type { PlayerState } from "../state.js";

export interface Sample {
  readonly at: string;
  readonly kind: string;
  readonly value: string;
  readonly peer: string;
}

export interface EdgeDetail {
  readonly from: string;
  readonly to: string;
  readonly kind: string;
  readonly recent: Sample[];
}

export interface NodeDetail {
  readonly name: string;
  readonly role: string;
  readonly type: string;
  readonly phase: string;
  readonly inputs: Sample[];
  readonly outputs: Sample[];
  readonly usage?: UsageRecord;
}

export class Details {
  #selected: string | null = null;

  constructor(
    private readonly title: HTMLElement,
    private readonly body: HTMLElement,
    private readonly base: URL,
  ) {}

  /** Returns to the whole session's figures. */
  clear(state: PlayerState): void {
    this.#selected = null;
    this.title.textContent = "usage";
    this.#renderSession(state);
  }

  /** Shows what one wire has carried. */
  async showEdge(from: string, to: string, kind: string): Promise<void> {
    this.#selected = `${from}→${to}`;
    this.title.textContent = `${from} → ${to}`;
    this.body.replaceChildren(text("p", "meta", "loading…"));

    try {
      const query = new URLSearchParams({ from, to, kind });
      const response = await fetch(new URL(`edge?${query}`, this.base));
      if (!response.ok) {
        this.body.replaceChildren(text("p", "meta", "no detail for this wire"));
        return;
      }

      const detail = (await response.json()) as EdgeDetail;
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
   * Repaints on a state change. A selected block is left alone: refetching it on
   * every event would flicker, and its figures are a moment's snapshot rather
   * than a live reading.
   */
  refresh(state: PlayerState): void {
    if (this.#selected) return;
    this.#renderSession(state);
  }

  #renderSession(state: PlayerState): void {
    const usage = state.usage;
    if (!usage) {
      this.body.replaceChildren(text("p", "meta", "nothing spent yet"));
      return;
    }

    const rows = usage.byModel.flatMap((record) => [
      text("span", "k", record.model),
      text("span", "v", `${amount(record)} · ${money(record.cost)}`),
    ]);

    const label = text("span", "k total", usage.complete ? "total" : "total (partly unpriced)");
    const total = text("span", "v total", usage.complete ? money(usage.totalCost) : `${money(usage.totalCost)}+`);

    const list = document.createElement("div");
    list.className = "kv";
    list.append(...rows, label, total);

    this.body.replaceChildren(list, text("p", "meta", "Click a block to read what it did."));
  }

  async #renderNode(name: string): Promise<void> {
    const response = await fetch(new URL(`nodes/${encodeURIComponent(name)}`, this.base));
    if (!response.ok) {
      this.body.replaceChildren(text("p", "meta", `no detail for ${name}`));
      return;
    }

    const detail = (await response.json()) as NodeDetail;
    const parts: HTMLElement[] = [facts(detail)];

    // A sink has no outputs and a source has no inputs, so neither is assumed.
    if (detail.outputs?.length) parts.push(samples("last outputs", detail.outputs));
    if (detail.inputs?.length) parts.push(samples("last inputs", detail.inputs));

    this.body.replaceChildren(...parts);
  }
}

function facts(detail: NodeDetail): HTMLElement {
  const rows: [string, string][] = [
    ["type", detail.type],
    ["role", detail.role],
    ["state", detail.phase],
  ];
  if (detail.usage?.model) rows.push(["model", detail.usage.model]);
  if (detail.usage) rows.push(["spent", `${amount(detail.usage)} · ${money(detail.usage.cost)}`]);

  const list = document.createElement("div");
  list.className = "kv";
  for (const [key, value] of rows) {
    list.append(text("span", "k", key), text("span", "v", value));
  }
  return list;
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

function amount(record: UsageRecord): string {
  const parts: string[] = [];
  if (record.images) parts.push(`${record.images} img`);
  if (record.audioSeconds) parts.push(`${record.audioSeconds.toFixed(0)}s`);
  const tokens = (record.inputTokens ?? 0) + (record.cachedInputTokens ?? 0) + (record.outputTokens ?? 0);
  if (tokens) parts.push(`${tokens} tok`);
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
