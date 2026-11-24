/**
 * The guardrail tab: the youth-protection rule the session's AI calls keep
 * to, where it came from and which blocks keep to it. It belongs to the whole
 * session, like usage, and to no block of the pipeline. Whoever may change it
 * (the engine says so with mayEdit) gets a box to try another one, and how
 * often each step repeats it: a lab experiment, from the next call on.
 */
import type { GuardrailReport, GuardrailSource, SessionRef } from "../client.js";
import { blockLabel, text as said, type TextKey } from "../texts.js";

const LEVELS: Record<string, TextKey> = {
  workshop: "guardrail.levelWorkshop",
  organisation: "guardrail.levelOrganisation",
  site13: "guardrail.levelSite13",
  site13p: "guardrail.levelSite13p",
  site18: "guardrail.levelSite18",
};

export class Guardrail {
  constructor(
    private readonly root: HTMLElement,
    private readonly session: SessionRef,
    /** A block the guardrail is kept to by was picked. */
    private readonly onBlock: (name: string) => void,
  ) {}

  /** Fetches the guardrail afresh and draws it. */
  async show(): Promise<void> {
    let report: GuardrailReport;
    try {
      ({ data: report } = await this.session.client.getApiEngineSessionsIdGuardrail(this.session.id));
    } catch {
      this.root.replaceChildren(line("p", "meta", said("guardrail.failed")));
      return;
    }
    this.#draw(report);
  }

  #draw(report: GuardrailReport): void {
    const parts: HTMLElement[] = [line("p", "meta", said("guardrail.intro"))];

    const current = document.createElement("section");
    const head = line("h4", "", said("insights.tabGuardrail"));
    if (report.edited) head.append(" ", line("span", "edited-mark", said("guardrail.edited")));
    current.append(head, line("p", "guardrail-text", report.text || said("guardrail.none")));
    parts.push(current);

    const facts = document.createElement("div");
    facts.className = "kv";
    facts.append(line("span", "k", said("guardrail.source")), line("span", "v", source(report.source)));
    if (report.edited) {
      facts.append(line("span", "k", said("guardrail.resolved")), line("span", "v", report.resolved || said("guardrail.none")));
    }
    if (report.readBy.length) {
      const blocks = document.createElement("span");
      blocks.className = "v guardrail-blocks";
      for (const name of report.readBy) {
        const open = document.createElement("button");
        open.type = "button";
        open.className = "problem-source";
        open.textContent = blockLabel(name);
        open.addEventListener("click", () => this.onBlock(name));
        blocks.append(open);
      }
      facts.append(line("span", "k", said("guardrail.readBy")), blocks);
    }
    parts.push(facts);

    if (report.mayEdit) {
      parts.push(this.#editor(report));
      if (report.frequencies.length) parts.push(this.#frequencies(report));
    }
    this.root.replaceChildren(...parts);
  }

  /** The box to try another guardrail, and the way back to the resolved one. */
  #editor(report: GuardrailReport): HTMLElement {
    const form = document.createElement("form");
    form.className = "injection";
    const box = document.createElement("textarea");
    box.rows = 4;
    box.value = report.text;
    box.setAttribute("aria-label", said("guardrail.edit"));
    const save = document.createElement("button");
    save.type = "submit";
    save.className = "injection-send";
    save.textContent = said("guardrail.save");
    const actions = document.createElement("div");
    actions.className = "prompt-actions";
    actions.append(save);
    if (report.edited) {
      const reset = document.createElement("button");
      reset.type = "button";
      reset.className = "injection-send";
      reset.textContent = said("guardrail.reset");
      reset.addEventListener("click", () => {
        box.value = report.resolved;
        form.requestSubmit();
      });
      actions.append(reset);
    }
    const status = line("p", "meta", said("guardrail.editHint"));
    form.append(line("span", "injection-title", said("guardrail.edit")), box, actions, status);

    form.addEventListener("submit", async (event) => {
      event.preventDefault();
      save.disabled = true;
      status.textContent = said("guardrail.saving");
      try {
        await this.session.client.putApiEngineSessionsIdGuardrail(this.session.id, { text: box.value });
        await this.show();
        const shown = this.root.querySelector<HTMLElement>(".injection .meta");
        if (shown) shown.textContent = said("guardrail.saved");
      } catch {
        status.textContent = said("guardrail.failed");
        save.disabled = false;
      }
    });
    return form;
  }

  /** How often each threaded step repeats the guardrail, each changeable. */
  #frequencies(report: GuardrailReport): HTMLElement {
    const section = document.createElement("section");
    section.append(line("h4", "", said("guardrail.frequency")), line("p", "meta", said("guardrail.frequencyHint")));
    for (const { block, frequency } of report.frequencies) {
      const form = document.createElement("form");
      form.className = "observer-beat";
      const label = line("span", "k", blockLabel(block));
      const field = document.createElement("input");
      field.type = "number";
      field.min = "1";
      field.max = "10";
      field.step = "1";
      field.value = String(frequency);
      field.setAttribute("aria-label", blockLabel(block));
      const set = document.createElement("button");
      set.type = "submit";
      set.textContent = said("guardrail.frequencySet");
      const status = line("span", "meta", "");
      form.append(label, field, set, status);
      form.addEventListener("submit", (event) => {
        event.preventDefault();
        void this.session.client
          .postApiEngineSessionsIdConstraintFrequency(this.session.id, { block, frequency: Number(field.value) })
          .then(() => (status.textContent = "✓"))
          .catch(() => (status.textContent = said("guardrail.frequencyFailed")));
      });
      section.append(form);
    }
    return section;
  }
}

/** Where a guardrail came from, in the reader's language. */
function source(from: GuardrailSource | undefined): string {
  const key = from && LEVELS[from.level];
  if (!key) return said("guardrail.sourceUnknown");
  return said(key).replace("{{name}}", from.name ?? "");
}

function line(tag: string, className: string, content: string): HTMLElement {
  const element = document.createElement(tag);
  if (className) element.className = className;
  element.textContent = content;
  return element;
}
