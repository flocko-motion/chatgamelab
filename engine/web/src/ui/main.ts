/**
 * Browser entry point. Its only jobs are constructing the headless player,
 * pointing a view at its state, and piping input into it.
 */
import { AudioIO } from "./audio.js";
import { Player } from "../player.js";
import { WebSocketTransport } from "../transport.js";
import { View } from "./view.js";
import { Details } from "./details.js";

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

const player = new Player(new WebSocketTransport(socketURL));
const typed = required<HTMLInputElement>("#typed");

const details = new Details(
  required<HTMLElement>("#details-title"),
  required<HTMLElement>("#details"),
  sessionBase,
);

const view = new View({
  status: required<HTMLElement>("#state"),
  props: required<HTMLElement>("#props"),
  details: required<HTMLElement>("#details"),
  detailsTitle: required<HTMLElement>("#details-title"),
  graph: required<HTMLElement>("#graph"),
  image: required<HTMLImageElement>("#scene"),
  feed: required<HTMLElement>("#feed"),
  talk: required<HTMLButtonElement>("#talk"),
  typed,
}, (name) => void details.show(name),
   () => details.clear(player.state));

player.subscribe((state) => {
  view.render(state);
  details.refresh(state);
});

// A reload has to rebuild what it missed before it starts listening, or the
// first live event would land on an empty page.
void player.restore(sessionBase).then(() => player.connect());

const audio = new AudioIO(new URL("./mic-worklet.js", import.meta.url).href);
const talk = required<HTMLButtonElement>("#talk");

talk.addEventListener("pointerdown", () => {
  player.takeFloor();
  void audio.startCapture((pcm) => player.speak(pcm));
});
const release = (): void => audio.stopCapture();
talk.addEventListener("pointerup", release);
talk.addEventListener("pointerleave", release);

// A live session accepts typed input as readily as spoken, which is how it is
// played without a microphone.
required<HTMLFormElement>("#compose").addEventListener("submit", (event) => {
  event.preventDefault();
  const text = typed.value.trim();
  if (!text) return;
  typed.value = "";
  view.echo(text);
  player.takeFloor();
  player.say(text);
});
