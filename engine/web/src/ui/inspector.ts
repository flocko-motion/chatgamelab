/**
 * The detail behind a block: what it is, what it is doing, what recently passed
 * through it and what it has cost.
 *
 * On a platform whose point is showing how the AI works, being able to click a
 * stage and read its actual inputs and outputs is the product rather than a
 * debug feature.
 */
import type { UsageRecord } from "../protocol.js";

export interface Sample {
  readonly kind: string;
  readonly value: string;
  readonly peer: string;
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

export class Inspector {
  constructor(
    private readonly dialog: HTMLDialogElement,
    private readonly body: HTMLElement,
    private readonly base: URL,
  ) {
    dialog.addEventListener("click", (event) => {
      // A click on the backdrop rather than the card closes it.
      if (event.target === dialog) dialog.close();
    });
  }

  async open(name: string): Promise<void> {
    this.body.replaceChildren(text("p", "meta", "loading…"));
    this.dialog.showModal();

    try {
      await this.#fill(name);
    } catch (error) {
      // Anything unhandled here used to leave the panel saying "loading…"
      // indefinitely, which reads as a hang rather than as a failure.
      this.body.replaceChildren(text("p", "meta", `could not read ${name}: ${String(error)}`));
    }
  }

  async #fill(name: string): Promise<void> {
    const response = await fetch(new URL(`nodes/${encodeURIComponent(name)}`, this.base));
    if (!response.ok) {
      this.body.replaceChildren(text("p", "meta", `no detail for ${name}`));
      return;
    }

    const detail = (await response.json()) as NodeDetail;
    const parts: HTMLElement[] = [
      text("h3", "", detail.name),
      facts(detail),
    ];

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
  if (detail.usage) rows.push(["spent", amountAndCost(detail.usage)]);

  const list = document.createElement("div");
  list.className = "kv";
  for (const [key, value] of rows) {
    list.append(text("span", "k", key), text("span", "v", value));
  }
  return list;
}

function samples(title: string, entries: Sample[]): HTMLElement {
  const section = document.createElement("section");
  section.append(text("h4", "", title));
  for (const entry of entries) {
    const row = document.createElement("div");
    row.className = "sample";
    row.append(
      text("span", "sample-kind", `${entry.kind} · ${entry.peer}`),
      // An empty value is the marker that closes an utterance, which is worth
      // naming rather than rendering as a blank line.
      text("span", "sample-value", entry.value || "(end of utterance)"),
    );
    section.append(row);
  }
  return section;
}

function amountAndCost(record: UsageRecord): string {
  const parts: string[] = [];
  if (record.images) parts.push(`${record.images} img`);
  if (record.audioSeconds) parts.push(`${record.audioSeconds.toFixed(0)}s`);
  const tokens = (record.inputTokens ?? 0) + (record.cachedInputTokens ?? 0) + (record.outputTokens ?? 0);
  if (tokens) parts.push(`${tokens} tok`);
  const amount = parts.join(" ") || "—";
  return record.cost ? `${amount} · $${record.cost.toFixed(4)}` : amount;
}

function text(tag: string, className: string, content: string): HTMLElement {
  const element = document.createElement(tag);
  if (className) element.className = className;
  element.textContent = content;
  return element;
}
