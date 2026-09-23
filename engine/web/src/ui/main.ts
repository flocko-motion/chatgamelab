/**
 * Browser entry point. Its only jobs are constructing the headless player,
 * pointing a view at its state, and piping the microphone into it.
 */
import { AudioIO } from "./audio.js";
import { Player } from "../player.js";
import { WebSocketTransport } from "../transport.js";
import { View } from "./view.js";

function required<T extends Element>(selector: string): T {
  const element = document.querySelector<T>(selector);
  if (!element) throw new Error(`missing element ${selector}`);
  return element;
}

// The session is whatever the server was started with; the player brings no
// spec of its own.
const sessionID = new URLSearchParams(location.search).get("session") ?? "standalone";

const socketURL = new URL(`../sessions/${sessionID}/live`, location.href);
socketURL.protocol = socketURL.protocol.replace("http", "ws");

const player = new Player(new WebSocketTransport(socketURL));
const view = new View({
  status: required<HTMLElement>("#state"),
  props: required<HTMLElement>("#props"),
  image: required<HTMLImageElement>("#scene"),
  feed: required<HTMLElement>("#feed"),
  talk: required<HTMLButtonElement>("#talk"),
});

player.subscribe((state) => view.render(state));

const audio = new AudioIO(new URL("./mic-worklet.js", import.meta.url).href);
const talk = required<HTMLButtonElement>("#talk");

talk.addEventListener("pointerdown", () => {
  player.takeFloor();
  void audio.startCapture((pcm) => player.speak(pcm));
});
const release = () => audio.stopCapture();
talk.addEventListener("pointerup", release);
talk.addEventListener("pointerleave", release);

player.connect();
