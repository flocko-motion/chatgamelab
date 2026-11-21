/**
 * Browser entry point. Its only jobs are constructing the headless playback-engine,
 * pointing a view at its state, and piping input into it.
 */
import { engineClient, liveURL, refusalOf, type SessionRef } from "../client.js";
import { PlaybackEngine } from "../playback-engine.js";
import {
  create,
  forget,
  opening,
  remembered,
  transientMemory,
  type Memory,
  type Opening,
  type Selector,
} from "../launch.js";
import { explain, holdsToTalk } from "../state.js";
import { notice, text, useLanguage, type Notice } from "../texts.js";
import { WebSocketTransport } from "../transport.js";
import { LaunchPanel } from "./launch.js";
import { AudioIO } from "./audio.js";
import { WebRTCVoice } from "./voice.js";
import { HoldToTalk } from "./talk.js";
import { mountSplit } from "./split.js";
import { View } from "./view.js";
import { Details } from "./details.js";
import { OptionsDialog } from "./options.js";
import { mountBackground } from "./theme/backgrounds/index.js";
import { PRESETS } from "./theme/presets/index.js";
import { textRenderer } from "./theme/text-effects/index.js";
import { fromGenerated, mergeTheme } from "./theme/resolve.js";

function required<T extends Element>(selector: string): T {
  const element = document.querySelector<T>(selector);
  if (!element) throw new Error(`missing element ${selector}`);
  return element;
}

// The engine is served from the page's origin, in the platform and in the dev server alike.
const client = engineClient(location.origin);
const params = new URLSearchParams(location.search);
const memories = { sessions: browserMemory(() => localStorage), pending: browserMemory(() => sessionStorage) };
const launch = new LaunchPanel({
  panel: required<HTMLElement>("#launch"),
  text: required<HTMLElement>("#launch-text"),
  actions: required<HTMLElement>("#launch-actions"),
  scene: required<HTMLElement>(".scene-area"),
});

// Before anything is said, so the first words are already in the player's language.
await useLanguage(params.get("lang") ?? navigator.language, location.href);

const opened = opening(params, memories);
const sessionID = await resolve(opened);
launch.hide();
if (params.get("session") !== sessionID) {
  // The address is now the session's, so a reload resumes it.
  history.replaceState(null, "", `?session=${encodeURIComponent(sessionID)}`);
}
if (opened.kind === "new" && window.parent !== window) {
  // Same origin only: the ID plays the session, so it goes to no other site.
  window.parent.postMessage({ type: "session", id: sessionID }, location.origin);
}
const session: SessionRef = { client, id: sessionID };

// The voice is the player's own connection to the model. It is handed to the
// core rather than opened here, because when it opens is the engine's call:
// nothing connects until the game has begun.
// Assigned once the view exists. The transport has to be built before the
// playback-engine, and the playback-engine before the view, so the one thing that closes the
// circle is handed over afterwards.
let recover = (): void => {};

// The speaker plays a spoken reply's audio as it arrives over the session's
// connection, as an Adventure's narration does; it also records push-to-talk.
const audio = new AudioIO(new URL("./mic-worklet.js", import.meta.url).href);
const playback = new PlaybackEngine(
  new WebSocketTransport(liveURL(session, location.origin), () => recover()),
  new WebRTCVoice(session),
  audio,
);
const typed = required<HTMLInputElement>("#typed");
const hold = required<HTMLButtonElement>("#hold");

const details = new Details(
  required<HTMLElement>("#details-title"),
  required<HTMLElement>("#details"),
  session,
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
  hold,
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
  onBackgroundClick: () => details.clear(playback.state),
  onEdgeClick: (from, to, kind) => void details.showEdge(from, to, kind),
});

// The game's theme applies once the session says which; the header offers the
// rest, where a teacher can try them.
const themes = required<HTMLSelectElement>("#theme");
themes.append(...Object.keys(PRESETS).sort().map((name) => new Option(name, name)));

function useTheme(preset: string): void {
  themes.value = preset;
  view.setTheme(mergeTheme(fromGenerated({ preset })), preset);
}

useTheme("default");
themes.addEventListener("change", () => useTheme(themes.value));

mountSplit(required<HTMLElement>("#split"));

const options = new OptionsDialog({
  form: required<HTMLFormElement>("#options"),
  offers: required<HTMLElement>("#options-offers"),
  problem: required<HTMLElement>("#options-problem"),
  start: required<HTMLButtonElement>("#options-start"),
  scene: required<HTMLElement>(".scene-area"),
  pictures: required<HTMLButtonElement>("#pictures"),
}, {
  onChoose: (offer, choice) => void playback.choose(session, offer, choice),
  onToggle: (toggle, on) => void playback.setToggle(session, toggle, on),
});

// A dropped turn's line goes back into the box once, unless the player has
// started typing something else, so sending it again is one press.
let givenBack: string | null = null;

// Hold to talk, or Space while not typing; release sends what was said.
const talk = new HoldToTalk(hold, audio, {
  onClip: (clip) => {
    view.awaitReply();
    playback.speak(clip);
  },
  onDenied: () => playback.fault(notice("errors.noMicrophone")),
}, () => document.activeElement === typed);

playback.subscribe((state) => {
  view.render(state);
  // Voice input went off, or the game cannot be played: a recording goes.
  if (!holdsToTalk(state) || hold.disabled) talk.cancel();
  options.render(state);
  details.refresh(state);
  if (state.redo !== givenBack) {
    givenBack = state.redo;
    if (givenBack !== null && !typed.value) typed.value = givenBack;
  }
});

// A dropped socket misses whatever was said while it was away, so coming back
// means rebuilding rather than carrying on: the same path a reload takes.
recover = () => {
  view.reset();
  void playback.rebuild(session);
};

// A reload has to rebuild what it missed before it starts listening, or the
// first live event would land on an empty page.
void playback.restore(session).then(async () => {
  const topology = playback.state.topology;
  if (topology?.theme && topology.theme in PRESETS) useTheme(topology.theme);
  if (topology?.language) {
    // The session's language is fixed at creation, and wins over the page's.
    document.documentElement.lang = topology.language;
    await useLanguage(topology.language, location.href);
    talk.refresh();
    view.render(playback.state);
  }
  playback.connect();
});

// One control for the whole conversation, because a live one has two states
// worth being in: open, where the character hears everything and the meter
// runs, and let go, which costs nothing and remembers everything. The view
// decides how it looks; this only says which way the press goes.
const pause = required<HTMLButtonElement>("#talk");
pause.addEventListener("click", () => {
  if (pause.disabled) return;
  if (playback.state.paused) playback.resume();
  else void playback.pause(session);
});

// What is visible is the view's business, decided from the wiring. This only
// has to say what happens when somebody uses it.
const compose = required<HTMLFormElement>("#compose");
compose.addEventListener("submit", (event) => {
  event.preventDefault();
  const text = typed.value.trim();
  if (!text || typed.disabled) return;
  typed.value = "";
  // The player's line comes back from the engine, on `said`.
  view.awaitReply();
  playback.say(text);
});

/**
 * Comes to a session: resumes the one named, or creates one for a game or a
 * share, offering first to continue the one this browser played last. Every
 * reason a game cannot start is said on the launch panel.
 */
async function resolve(open: Opening): Promise<string> {
  switch (open.kind) {
    case "nothing":
      return launch.ask(text("launch.noGame"), []);
    case "resume": {
      const refused = await probe(open.id);
      if (!refused) return open.id;
      if (!open.selector) return launch.ask(text(refused.key), []);
      await launch.ask(text(refused.key), [text("launch.startAnew")]);
      forget(memories.sessions, open.selector);
      return start(open.selector);
    }
    case "new": {
      const prior = remembered(memories.sessions, open.selector);
      if (!prior) return start(open.selector);
      const answer = await launch.ask(text("launch.inProgress"), [text("launch.continue"), text("launch.startAnew")]);
      if (answer === text("launch.continue")) return resolve({ kind: "resume", id: prior, selector: open.selector });
      forget(memories.sessions, open.selector);
      return start(open.selector);
    }
  }
}

/** Creates a session, as often as the player tries again. */
async function start(selector: Selector): Promise<string> {
  const language = params.get("lang") ?? navigator.language;
  for (;;) {
    launch.wait(text("launch.starting"));
    const created = await create(client, selector, language, memories);
    if ("id" in created) return created.id;
    await launch.ask(text(created.notice.key), [text("launch.tryAgain")]);
  }
}

/** Whether a session can be had, and if not, why; reaching it restores it. */
async function probe(id: string): Promise<Notice | null> {
  try {
    await client.getApiEngineSessionsIdTopology(id);
    return null;
  } catch (thrown) {
    const refusal = await refusalOf(thrown);
    return refusal ? explain(refusal) : notice("errors.serverUnreachable");
  }
}

/** Browser storage, which a private window or a blocked site may refuse. */
function browserMemory(storage: () => Storage): Memory {
  try {
    const held = storage();
    void held.length;
    return {
      get: (key) => {
        try {
          return held.getItem(key);
        } catch {
          return null;
        }
      },
      set: (key, value) => {
        try {
          held.setItem(key, value);
        } catch {
          // Kept nowhere: the page still plays, and forgets.
        }
      },
      remove: (key) => {
        try {
          held.removeItem(key);
        } catch {
          // Nothing kept, nothing to remove.
        }
      },
      keys: () => {
        try {
          return Array.from({ length: held.length }, (_, i) => held.key(i) ?? "");
        } catch {
          return [];
        }
      },
    };
  } catch {
    return transientMemory();
  }
}
