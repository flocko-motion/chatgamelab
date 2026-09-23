/**
 * Dumb visualisation: paints whatever the state says and decides nothing.
 * Layout follows the old player — status line, scene image, transcript feed,
 * controls at the top — without any of its theming.
 */
import type { PlayerState } from "../state.js";

export class View {
  #renderedUtterances = 0;
  #renderedFlags = 0;
  #renderedNotes = 0;
  #shownImage: string | null = null;

  constructor(
    private readonly elements: {
      status: HTMLElement;
      props: HTMLElement;
      image: HTMLImageElement;
      feed: HTMLElement;
      talk: HTMLButtonElement;
    },
  ) {}

  render(state: PlayerState): void {
    this.elements.status.textContent = state.connection;
    this.elements.talk.disabled = state.connection !== "open";

    const props = Object.entries(state.props);
    this.elements.props.textContent = props.map(([k, v]) => `${k}: ${v}`).join("   ");

    if (state.image && state.image !== this.#shownImage) {
      this.#shownImage = state.image;
      // A placeholder adapter returns text rather than bytes, so show whichever
      // arrived instead of a broken image.
      if (state.image.startsWith("data:") || state.image.startsWith("http")) {
        this.elements.image.src = state.image;
        this.elements.image.hidden = false;
      } else {
        this.#append("meta", state.image);
      }
    }

    // Append-only: the feed is a timeline, so it is extended rather than
    // rebuilt from state on every change.
    for (let i = this.#renderedUtterances; i < state.utterances.length; i++) {
      const utterance = state.utterances[i];
      if (!utterance) continue;
      this.#append("npc", utterance.text).dataset["utterance"] = String(utterance.id);
    }
    this.#renderedUtterances = state.utterances.length;

    // The open utterance grows in place as deltas arrive.
    const last = state.utterances.at(-1);
    if (last) {
      const node = this.elements.feed.querySelector<HTMLElement>(
        `[data-utterance="${last.id}"]`,
      );
      if (node) node.textContent = last.text;
    }

    for (let i = this.#renderedFlags; i < state.flags.length; i++) {
      this.#append("flag", `observer: ${state.flags[i]}`);
    }
    this.#renderedFlags = state.flags.length;

    for (let i = this.#renderedNotes; i < state.notes.length; i++) {
      this.#append("meta", String(state.notes[i]));
    }
    this.#renderedNotes = state.notes.length;
  }

  #append(className: string, text: string): HTMLElement {
    const element = document.createElement("div");
    element.className = `line ${className}`;
    element.textContent = text;
    this.elements.feed.append(element);
    this.elements.feed.scrollTop = this.elements.feed.scrollHeight;
    return element;
  }
}
