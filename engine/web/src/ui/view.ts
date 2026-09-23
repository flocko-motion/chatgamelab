/**
 * Dumb visualisation: paints whatever the state says and decides nothing.
 *
 * Split screen — the game on the left, drawn the way the platform's player
 * draws it and in the game's theme, and the instrumentation on the right. What
 * the engine is doing and what it costs are part of the subject on this
 * platform, so they get a panel rather than a developer console.
 */
import {
  acceptsTypedInput,
  audioState,
  hasStandingImage,
  voiceMode,
  type AudioState,
  type PlayerState,
} from "../state.js";
import { Flowchart } from "./flowchart.js";
import { applyTheme } from "./theme/apply.js";
import { mergeTheme, statusEmoji } from "./theme/resolve.js";
import type { BackgroundAnimation, GameTheme, TextEffect, TextEffectScope } from "./theme/types.js";

export interface ViewElements {
  root: HTMLElement;
  title: HTMLElement;
  props: HTMLElement;
  graph: HTMLElement;
  background: HTMLElement;
  scroll: HTMLElement;
  feed: HTMLElement;
  thinking: HTMLElement;
  thinkingText: HTMLElement;
  /** The one control a live conversation has: pause it, or pick it up again. */
  talk: HTMLButtonElement;
  /** What it holds, so the state has a word as well as a shape. */
  audio: HTMLElement;
  audioLabel: HTMLElement;
  typed: HTMLInputElement;
  /** The whole input bar, which only exists once the game can take input. */
  compose: HTMLElement;
  /** The typed half of it, for genres that have somewhere to put words. */
  inputs: HTMLElement;
  /** Stands in the bar's place while there is nothing anybody can do with it. */
  notice: HTMLElement;
  noticeText: HTMLElement;
  /** Holds the one picture a session has, above the conversation. */
  standing: HTMLElement;
  /** The area the portrait, the conversation and the controls share. */
  sceneArea: HTMLElement;
  /** The pinned controls, whose height the conversation makes room for. */
  dock: HTMLElement;
  send: HTMLButtonElement;
  lightbox: HTMLElement;
}

/** Paints one element's text with an effect, as the text grows. */
export interface TextRenderer {
  update(text: string): void;
  dispose(): void;
}

/** What draws a theme's moving parts, handed in so the view holds no animation code. */
export interface ThemeRenderers {
  text(effect: TextEffect, element: HTMLElement): TextRenderer;
  /** Starts an animation in the layer and returns what stops it. */
  background(layer: HTMLElement, animation: BackgroundAnimation): () => void;
}

export interface ViewHandlers {
  onNodeClick(name: string): void;
  onBackgroundClick(): void;
  onEdgeClick(from: string, to: string, kind: string): void;
}

type Scope = keyof TextEffectScope;

/** What the audio control says the conversation is doing. */
const AUDIO_LABEL: Record<AudioState, string> = {
  live: "Listening",
  pausing: "Pausing…",
  paused: "Paused",
  resuming: "Resuming…",
};

/** What pressing it would do, which is what a tooltip and a screen reader want. */
const AUDIO_ACTION: Record<AudioState, string> = {
  live: "Pause the conversation",
  pausing: "Pausing the conversation",
  paused: "Resume the conversation",
  resuming: "Resuming the conversation",
};

/**
 * Why the game cannot be played, and whether that is a wait or a stop.
 *
 * The difference is the whole of what a spinner promises. One says something is
 * on its way and the page is still in the game; the other says nothing more is
 * coming, and turning a wheel beside it would be a lie told once a second.
 */
interface Unavailable {
  readonly text: string;
  readonly waiting: boolean;
}

const waiting = (text: string): Unavailable => ({ text, waiting: true });
const stopped = (text: string): Unavailable => ({ text, waiting: false });

/** A piece of text on screen, and whatever is painting it. */
interface Painted {
  readonly element: HTMLElement;
  readonly scope: Scope;
  text: string;
  renderer: TextRenderer;
}

export class View {
  #renderedUtterances = 0;
  #renderedNotes = 0;
  #shownImage: string | null = null;
  #shownProps: Record<string, string> | null = null;
  #cards = new Map<number, HTMLElement>();
  #narratives = new Map<number, Painted>();
  #painted: Painted[] = [];
  #fields: Painted[] = [];
  /** A picture that arrived before the line it illustrates. */
  #pendingImage: string | null = null;
  /** Set once the player has spoken or typed, until the reply begins. */
  #awaiting = false;
  #theme: GameTheme = mergeTheme(undefined);
  #stopBackground: () => void = () => {};
  #last: PlayerState | null = null;
  #flowchart: Flowchart;

  constructor(
    private readonly elements: ViewElements,
    private readonly renderers: ThemeRenderers,
    handlers: ViewHandlers,
  ) {
    this.#flowchart = new Flowchart(
      elements.graph,
      handlers.onNodeClick,
      handlers.onBackgroundClick,
      handlers.onEdgeClick,
    );
    elements.lightbox.addEventListener("click", () => (elements.lightbox.hidden = true));
    this.#watchDock();
    this.setTheme(this.#theme);
  }

  /**
   * Restyles the game half. Colours and shapes change in place; text effects
   * and the background animation are restarted in the new theme's.
   */
  setTheme(theme: GameTheme, preset = "default"): void {
    this.#theme = theme;
    applyTheme(this.elements.root, theme, preset);

    this.#stopBackground();
    this.elements.background.replaceChildren();
    this.#stopBackground = this.renderers.background(
      this.elements.background,
      theme.background.animation ?? "none",
    );

    for (const painted of this.#painted) {
      painted.renderer.dispose();
      painted.renderer = this.#renderer(painted.scope, painted.element);
      painted.renderer.update(painted.text);
    }

    this.#shownProps = null;
    if (this.#last) this.render(this.#last);
  }

  render(state: PlayerState): void {
    this.#last = state;

    // A reason the game cannot be played stands where the controls would be,
    // inside the bar rather than in place of it. Every one of them means the
    // controls would not work, and a control that is there and useless invites
    // a try and then refuses it — but the bar itself is the one fixed thing at
    // the foot of the scene, and swapping it out moves everything above it.
    const blocked = this.#unavailable(state);
    show(this.elements.notice, blocked !== null);
    if (blocked !== null) {
      this.elements.noticeText.textContent = blocked.text;
      this.elements.notice.classList.toggle("waiting", blocked.waiting);
    }

    const audio = audioState(state);
    // The bar holds one thing at a time: the controls, or the reason there are
    // none.
    this.#renderAudio(blocked === null ? audio : null);

    // The text box appears only if the wiring has somewhere to deliver what is
    // typed. A live conversation has no node that takes words, because the
    // model has no event that would carry them; a conversation that has been
    // let go has no session to take them either.
    const typing =
      blocked === null && acceptsTypedInput(state) && (audio === null || audio === "live");
    show(this.elements.inputs, typing);
    // With nothing to type into, the button is the bar's whole content and
    // belongs in the middle of it.
    this.elements.compose.classList.toggle("no-typing", !typing);
    this.elements.typed.disabled = !typing;
    this.elements.send.disabled = !typing;

    // The game's own name, which is what somebody came to play. The wiring's
    // name is the genre, and stands in only until an author has given one.
    this.elements.title.textContent = state.topology?.title || state.topology?.name || "…";

    this.#renderGraph(state);
    this.#renderProps(state);
    this.#renderFeed(state);
    this.#renderImage(state);
    this.#renderThinking(state, blocked);
  }

  /**
   * The one control a live conversation has, and the one place its state is
   * said.
   *
   * The icon is which state the conversation is in — a microphone while the
   * character can hear you, pause bars once it has been let go — rather than
   * which button this is, because the first question a player has is whether
   * they are being heard.
   */
  #renderAudio(audio: AudioState | null): void {
    show(this.elements.audio, audio !== null);
    if (!audio) return;

    this.elements.audio.dataset.state = audio;
    this.elements.audioLabel.textContent = AUDIO_LABEL[audio];
    // A transition is not something to press again: the second press would race
    // the answer to the first, and both states it could land in are wrong.
    this.elements.talk.disabled = audio === "pausing" || audio === "resuming";
    this.elements.talk.setAttribute("aria-label", AUDIO_ACTION[audio]);
    this.elements.talk.title = AUDIO_ACTION[audio];
  }

  /**
   * Forgets the drawn conversation, for a rebuild after a reconnection. The
   * core clears its side; this clears what was painted from it, so the two do
   * not disagree about how much has already been shown.
   */
  reset(): void {
    this.elements.feed.replaceChildren();
    this.#cards.clear();
    this.#narratives.clear();
    this.#renderedUtterances = 0;
    this.#renderedNotes = 0;
    this.#shownImage = null;
    this.#pendingImage = null;
  }

  /** Echoes what the player typed, which the engine never sends back. */
  echo(text: string): void {
    const action = element("div", "player-action");
    const bubble = element("div", "bubble");
    const words = element("span", "bubble-text");
    bubble.append(words);
    action.append(bubble);
    this.#paint(words, "playerMessages", text);
    this.#append(action);
    this.awaitReply();
  }

  /** Shows that a reply is coming, from the moment the player hands over. */
  awaitReply(): void {
    this.#awaiting = true;
    this.elements.thinking.hidden = false;
    this.#scrollToEnd();
  }

  #renderer(scope: Scope, target: HTMLElement): TextRenderer {
    const enabled = this.#theme.gameMessage.textEffectScope[scope];
    return this.renderers.text(enabled ? this.#theme.gameMessage.textEffect : "none", target);
  }

  /** Paints text into an element and keeps it, so a new theme can repaint it. */
  #paint(target: HTMLElement, scope: Scope, text: string): Painted {
    const painted: Painted = { element: target, scope, text, renderer: this.#renderer(scope, target) };
    painted.renderer.update(text);
    this.#painted.push(painted);
    return painted;
  }

  #renderGraph(state: PlayerState): void {
    if (!state.topology) return;
    this.#flowchart.draw(state.topology);
    this.#flowchart.update(state.phases, state.usage?.byNode ?? []);
  }

  #renderProps(state: PlayerState): void {
    // Rebuilt only when the status itself changes: every audio chunk renders,
    // and rebuilding on each one would restart the pulse on a changed value.
    if (state.props === this.#shownProps) return;
    const previous = this.#shownProps ?? {};
    const first = this.#shownProps === null;
    this.#shownProps = state.props;

    for (const field of this.#fields) {
      field.renderer.dispose();
      this.#painted.splice(this.#painted.indexOf(field), 1);
    }
    this.#fields = [];

    const entries = Object.entries(state.props);
    // Hidden until a genre tracks something, since a live conversation tracks
    // nothing.
    this.elements.props.hidden = entries.length === 0;
    this.elements.props.replaceChildren(
      ...entries.map(([key, value]) => {
        const field = element("div", "field");
        if (!first && previous[key] !== undefined && previous[key] !== value) {
          field.classList.add("changed");
        }
        const name = element("span", "field-name");
        const emoji = statusEmoji(this.#theme, key);
        if (emoji) name.append(element("span", "field-emoji", emoji));
        const label = element("span", "");
        const amount = element("span", "field-value");
        name.append(label);
        field.append(name, amount);
        this.#fields.push(this.#paint(label, "statusFields", `${key}:`));
        this.#fields.push(this.#paint(amount, "statusFields", value));
        return field;
      }),
    );
  }

  #renderImage(state: PlayerState): void {
    if (!state.image || state.image === this.#shownImage) return;
    this.#shownImage = state.image;

    // One picture for the whole session is the character being spoken to, not
    // an illustration of a moment. It is held above the conversation so it
    // stays in view, where the timeline would have scrolled it away.
    if (hasStandingImage(state)) {
      this.#standing(state.image);
      return;
    }

    if (!state.image.startsWith("data:") && !state.image.startsWith("http")) {
      // A placeholder adapter returns text where a real one returns bytes.
      this.#append(element("div", "system-message", state.image));
      return;
    }

    // A picture belongs to the scene it was made for: the latest one if that
    // has none yet, otherwise the next one — a portrait made during preparation
    // arrives before the character has said anything.
    const last = this.#cards.get(state.utterances.at(-1)?.id ?? -1);
    if (last && !last.querySelector(".scene-image")) this.#illustrate(last, state.image);
    else this.#pendingImage = state.image;
  }

  /** Puts the session's one picture in the frame that holds it. */
  #standing(src: string): void {
    const frame = this.elements.standing;
    if (!frame) return;

    const picture = element("img", "") as HTMLImageElement;
    picture.src = src;
    picture.alt = "";
    frame.replaceChildren(picture);
    frame.hidden = false;
    this.elements.sceneArea.style.setProperty("--standing-height", "30vh");
  }

  /**
   * Keeps the conversation clear of the controls beneath it.
   *
   * Measured rather than assumed, because the bar changes height with what it
   * offers: a text box, a microphone, a resume button, and a status line above
   * them that comes and goes.
   */
  #watchDock(): void {
    const set = (): void => {
      const height = this.elements.dock.offsetHeight;
      this.elements.sceneArea.style.setProperty("--dock-height", `${height}px`);
    };
    set();
    new ResizeObserver(set).observe(this.elements.dock);
  }

  #renderFeed(state: PlayerState): void {
    // Append-only: the feed is a timeline, so it is extended rather than
    // rebuilt on every change.
    for (let i = this.#renderedUtterances; i < state.utterances.length; i++) {
      const utterance = state.utterances[i];
      if (!utterance) continue;
      if (this.elements.feed.childElementCount > 0) this.#append(divider());
      const card = sceneCard();
      this.#cards.set(utterance.id, card);
      const narrative = card.querySelector<HTMLElement>(".narrative");
      if (narrative) this.#narratives.set(utterance.id, this.#paint(narrative, "gameMessages", ""));
      if (this.#pendingImage) {
        this.#illustrate(card, this.#pendingImage);
        this.#pendingImage = null;
      }
      this.#append(card);
    }
    if (state.utterances.length > this.#renderedUtterances) this.#awaiting = false;
    this.#renderedUtterances = state.utterances.length;

    // Only the newest two can still be changing: the open utterance grows in
    // place, and the one before it may have just been closed. A turn-based
    // genre never closes its last line, so a line counts as being spoken only
    // while something in the graph is still working.
    const working = Object.values(state.phases).includes("working");
    for (const utterance of state.utterances.slice(-2)) {
      const card = this.#cards.get(utterance.id);
      const narrative = this.#narratives.get(utterance.id);
      if (!card || !narrative) continue;
      card.querySelector(".game-scene")?.classList.toggle("streaming", working && !utterance.complete);
      if (narrative.text !== utterance.text) {
        narrative.text = utterance.text;
        narrative.renderer.update(utterance.text);
        this.#scrollToEnd();
      }
    }

    // Nothing but the conversation reaches this feed. A block that corrects the
    // character is wired to the character, not to here, so what it found has
    // nowhere to arrive and nothing to render.

    for (let i = this.#renderedNotes; i < state.notes.length; i++) {
      this.#append(element("div", "system-message", String(state.notes[i])));
    }
    this.#renderedNotes = state.notes.length;
  }

  /**
   * Says that a reply is on its way, immediately above the input bar. That is
   * where somebody about to act is already looking; a state shown in a corner
   * of the instrumentation is a state nobody reads.
   *
   * A fault that leaves the controls working is said here too, for the same
   * reason and in the same place. One that does not is a notice instead, and
   * this stays out of its way.
   */
  #renderThinking(state: PlayerState, blocked: Unavailable | null): void {
    const fault = blocked === null ? state.problem : null;
    this.elements.thinking.hidden = blocked !== null || !(fault || this.#awaiting);
    this.elements.thinking.classList.toggle("fault", fault !== null);
    this.elements.thinkingText.textContent = fault ?? this.#theme.thinking.text;
  }

  /**
   * Why the game cannot be played right now, in words a player can act on, or
   * null while it can be.
   *
   * Everything here replaces the input bar, so the test for belonging is
   * whether the controls would work: a conversation that has been let go is not
   * on this list, because picking it up is exactly what the microphone is for.
   */
  #unavailable(state: PlayerState): Unavailable | null {
    const audio = audioState(state);
    // A fault outranks whatever the game would otherwise be waiting for — but
    // not a control that is still there to press. A resume that failed says so
    // beside the microphone rather than taking away the one thing that would
    // fix it.
    if (state.problem && audio === null) return stopped(state.problem);

    // The socket's own trouble is said here, because here is the only place
    // anything is said: a player watching for their turn is looking at the
    // input bar, and a page that goes quiet without saying why reads as a game
    // that has stopped listening.
    switch (state.connection) {
      case "idle":
      case "connecting":
        return waiting("Connecting…");
      case "reconnecting":
        return waiting("Reconnecting…");
      case "failed":
        return stopped("Lost the connection to the game. Reload to try again.");
      case "closed":
        return stopped("The game has ended.");
    }
    if (!state.started) return waiting("Setting the scene…");
    // A live genre is not playable until the player's own audio connection is
    // up, which takes a moment and can fail on its own. Once there is a control
    // for it, the control says so instead.
    if (voiceMode(state) === "full-duplex" && audio === null) return waiting("Waking the character…");
    return null;
  }

  #illustrate(card: HTMLElement, src: string): void {
    const frame = element("button", "scene-image") as HTMLButtonElement;
    frame.type = "button";
    frame.setAttribute("aria-label", "Enlarge the picture");
    const image = document.createElement("img");
    image.src = src;
    image.alt = "Scene illustration";
    frame.append(image);
    frame.addEventListener("click", () => this.#enlarge(src));

    const scene = card.querySelector(".game-scene");
    scene?.classList.add("with-image");
    scene?.querySelector(".narrative")?.before(frame);
  }

  #enlarge(src: string): void {
    const image = this.elements.lightbox.querySelector("img");
    if (image) image.src = src;
    this.elements.lightbox.hidden = false;
  }

  #append(child: HTMLElement): void {
    this.elements.feed.append(child);
    this.#scrollToEnd();
  }

  #scrollToEnd(): void {
    this.elements.scroll.scrollTop = this.elements.scroll.scrollHeight;
  }
}

/** A scene card. Its four corners are always there; the theme decides which show. */
function sceneCard(): HTMLElement {
  const card = element("article", "scene-card");
  const scene = element("div", "game-scene");
  for (const corner of ["tl", "tr", "bl", "br"]) {
    scene.append(element("span", `corner ${corner}`));
  }
  scene.append(element("div", "narrative"));
  card.append(scene);
  return card;
}

function divider(): HTMLElement {
  const line = element("div", "divider");
  line.setAttribute("aria-hidden", "true");
  line.append(element("span", "divider-symbol"));
  return line;
}

function element(tag: string, className: string, text?: string): HTMLElement {
  const node = document.createElement(tag);
  if (className) node.className = className;
  if (text !== undefined) node.textContent = text;
  return node;
}

/**
 * Shows or hides an element with an inline style rather than the `hidden`
 * attribute, because a stylesheet rule giving it a display wins over `hidden`
 * and the control comes back.
 */
function show(element: HTMLElement, visible: boolean): void {
  element.style.display = visible ? "" : "none";
}
