/**
 * Browser entry point. Its only jobs are constructing the headless player,
 * pointing a view at its state, and piping input into it.
 */
import { Player } from "../player.js";
import { WebSocketTransport } from "../transport.js";
import { WebRTCVoice } from "./voice.js";
import { mountSplit } from "./split.js";
import { View } from "./view.js";
import { Details } from "./details.js";
import { mountBackground } from "./theme/backgrounds/index.js";
import { PRESETS } from "./theme/presets/index.js";
import { textRenderer } from "./theme/text-effects/index.js";
import { fromGenerated, mergeTheme } from "./theme/resolve.js";

function required<T extends Element>(selector: string): T {
  const element = document.querySelector<T>(selector);
  if (!element) throw new Error(`missing element ${selector}`);
  return element;
}

// The session is whatever the server was started with; the player brings no
// spec of its own.
const sessionID = new URLSearchParams(location.search).get("session") ?? "standalone";
const sessionBase = new URL(`../sessions/${sessionID}/`, location.href);

const socketURL = new URL("live", sessionBase);
socketURL.protocol = socketURL.protocol.replace("http", "ws");

// The voice is the player's own connection to the model. It is handed to the
// core rather than opened here, because when it opens is the engine's call:
// nothing connects until the game has begun.
// Assigned once the view exists. The transport has to be built before the
// player, and the player before the view, so the one thing that closes the
// circle is handed over afterwards.
let recover = (): void => {};

const player = new Player(
  new WebSocketTransport(socketURL, () => recover()),
  new WebRTCVoice(sessionBase),
);
const typed = required<HTMLInputElement>("#typed");

const details = new Details(
  required<HTMLElement>("#details-title"),
  required<HTMLElement>("#details"),
  sessionBase,
);

const view = new View({
  root: required<HTMLElement>("main"),
  title: required<HTMLElement>("#title"),
  props: required<HTMLElement>("#props"),
  graph: required<HTMLElement>("#graph"),
  background: required<HTMLElement>("#background"),
  scroll: required<HTMLElement>("#scroll"),
  feed: required<HTMLElement>("#feed"),
  thinking: required<HTMLElement>("#thinking"),
  thinkingText: required<HTMLElement>("#thinking-text"),
  talk: required<HTMLButtonElement>("#talk"),
  audio: required<HTMLElement>("#audio"),
  audioLabel: required<HTMLElement>("#audio-label"),
  typed,
  compose: required<HTMLElement>("#compose"),
  inputs: required<HTMLElement>(".input-wrap"),
  notice: required<HTMLElement>("#notice"),
  noticeText: required<HTMLElement>("#notice-text"),
  standing: required<HTMLElement>("#standing"),
  sceneArea: required<HTMLElement>(".scene-area"),
  dock: required<HTMLElement>("#dock"),
  send: required<HTMLButtonElement>("#send"),
  lightbox: required<HTMLElement>("#lightbox"),
}, {
  text: textRenderer,
  background: mountBackground,
}, {
  onNodeClick: (name) => void details.show(name),
  onBackgroundClick: () => details.clear(player.state),
  onEdgeClick: (from, to, kind) => void details.showEdge(from, to, kind),
});

// The engine does not choose a theme yet, so the page takes a preset from the
// URL and offers the rest in the header, where a teacher can try them.
const themes = required<HTMLSelectElement>("#theme");
const chosen = new URLSearchParams(location.search).get("theme") ?? "default";
themes.append(...Object.keys(PRESETS).sort().map((name) => new Option(name, name, false, name === chosen)));

function useTheme(preset: string): void {
  view.setTheme(mergeTheme(fromGenerated({ preset })), preset);
  const url = new URL(location.href);
  url.searchParams.set("theme", preset);
  history.replaceState(null, "", url);
}

useTheme(themes.value);
themes.addEventListener("change", () => useTheme(themes.value));

mountSplit(required<HTMLElement>("#split"));

player.subscribe((state) => {
  view.render(state);
  details.refresh(state);
});

// A dropped socket misses whatever was said while it was away, so coming back
// means rebuilding rather than carrying on: the same path a reload takes.
recover = () => {
  view.reset();
  void player.rebuild(sessionBase);
};

// A reload has to rebuild what it missed before it starts listening, or the
// first live event would land on an empty page.
void player.restore(sessionBase).then(() => player.connect());

// One control for the whole conversation, because a live one has two states
// worth being in: open, where the character hears everything and the meter
// runs, and let go, which costs nothing and remembers everything. The view
// decides how it looks; this only says which way the press goes.
const talk = required<HTMLButtonElement>("#talk");
talk.addEventListener("click", () => {
  if (talk.disabled) return;
  if (player.state.paused) player.resume();
  else void player.pause(sessionBase);
});

// What is visible is the view's business, decided from the wiring. This only
// has to say what happens when somebody uses it.
const compose = required<HTMLFormElement>("#compose");
compose.addEventListener("submit", (event) => {
  event.preventDefault();
  const text = typed.value.trim();
  if (!text || typed.disabled) return;
  typed.value = "";
  view.echo(text);
  player.takeFloor();
  player.say(text);
});
