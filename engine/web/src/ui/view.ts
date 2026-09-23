/**
 * Dumb visualisation: paints whatever the state says and decides nothing.
 *
 * Split screen — the conversation on the left, the instrumentation on the
 * right. What the engine is doing and what it costs are part of the subject on
 * this platform, so they get a panel rather than a developer console.
 */
import type { PlayerState } from "../state.js";
import { Flowchart } from "./flowchart.js";

export interface ViewElements {
  status: HTMLElement;
  props: HTMLElement;
  details: HTMLElement;
  detailsTitle: HTMLElement;
  graph: HTMLElement;
  image: HTMLImageElement;
  feed: HTMLElement;
  talk: HTMLButtonElement;
  typed: HTMLInputElement;
}

export class View {
  #renderedUtterances = 0;
  #renderedFlags = 0;
  #renderedNotes = 0;
  #shownImage: string | null = null;
  #flowchart: Flowchart;

  constructor(
    private readonly elements: ViewElements,
    onNodeClick: (name: string) => void,
    onBackgroundClick: () => void,
  ) {
    this.#flowchart = new Flowchart(elements.graph, onNodeClick, onBackgroundClick);
  }

  render(state: PlayerState): void {
    // Playable, not merely connected: the gate holds the player's inputs until
    // the init stem has finished, so the controls follow the gate rather than
    // the socket.
    const playable = state.connection === "open" && state.started;
    this.elements.status.textContent = this.#statusLabel(state);
    this.elements.talk.disabled = !playable;
    this.elements.typed.disabled = !playable;

    this.#renderGraph(state);
    this.#renderProps(state);
    this.#renderImage(state);
    this.#renderFeed(state);
  }

  #statusLabel(state: PlayerState): string {
    if (state.connection !== "open") return state.connection;
    return state.started ? "ready" : "preparing…";
  }

  /** Echoes what the player typed, which the engine never sends back. */
  echo(text: string): void {
    this.#append("you", text);
  }

  #renderGraph(state: PlayerState): void {
    if (!state.topology) return;
    this.#flowchart.draw(state.topology);
    this.#flowchart.update(state.phases, state.usage?.byNode ?? []);
  }

  #renderProps(state: PlayerState): void {
    const entries = Object.entries(state.props);
    // Status belongs with the game rather than with the instrumentation: it is
    // what a player reads, not what an observer inspects. Hidden until a genre
    // tracks something, since a live conversation tracks nothing.
    this.elements.props.hidden = entries.length === 0;
    this.elements.props.replaceChildren(
      ...entries.map(([key, value]) => span("field", `${key} ${value}`)),
    );
  }

  #renderImage(state: PlayerState): void {
    if (!state.image || state.image === this.#shownImage) return;
    this.#shownImage = state.image;
    if (state.image.startsWith("data:") || state.image.startsWith("http")) {
      this.elements.image.src = state.image;
      this.elements.image.hidden = false;
    } else {
      // A placeholder adapter returns text where a real one returns bytes.
      this.#append("meta", state.image);
    }
  }

  #renderFeed(state: PlayerState): void {
    // Append-only: the feed is a timeline, so it is extended rather than
    // rebuilt on every change.
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

function span(className: string, text: string): HTMLElement {
  const element = document.createElement("span");
  element.className = className;
  element.textContent = text;
  return element;
}
