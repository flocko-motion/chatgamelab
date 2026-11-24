/**
 * Browser entry point. Its only jobs are constructing the headless playback-engine,
 * pointing a view at its state, and piping input into it.
 */
import { engineClient, liveURL, refusalOf, type SessionRef } from "../client.js";
import { PlaybackEngine } from "../playback-engine.js";
import { create, opening, transientMemory, type Memory, type Opening, type Selector } from "../launch.js";
import { explain, holdsToTalk } from "../state.js";
import { notice, text, useLanguage, type Notice } from "../texts.js";
import { TappedTransport, WebSocketTransport } from "../transport.js";
import { LaunchPanel } from "./launch.js";
import { mountEmbed } from "./embed.js";
import { sessionAddress } from "../embed.js";
import { AudioIO } from "./audio.js";
import { SeatedVoice } from "./voice.js";
import { HoldToTalk } from "./talk.js";
import { Collapsibles } from "./collapse.js";
import { mountInsightsSplit, mountSplit } from "./split.js";
import { View } from "./view.js";
import { Details } from "./details.js";
import { startDebugging, type Debugging } from "./debug.js";
import { Insights } from "./insights.js";
import { InsightsTabs } from "./tabs.js";
import { Guardrail } from "./guardrail.js";
import { Problems } from "./problems.js";
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
// Framed by another website: the plain game, and the host's CSS.
mountEmbed(params);
// Only the one-time key of a creation under way, for this tab: sessions are
// remembered by whoever opens the page, the platform or the address.
const pending = browserMemory(() => sessionStorage);
const launch = new LaunchPanel({
  panel: required<HTMLElement>("#launch"),
  text: required<HTMLElement>("#launch-text"),
  actions: required<HTMLElement>("#launch-actions"),
  scene: required<HTMLElement>(".scene-area"),
});

// Before anything is said, so the first words are already in the player's language.
await useLanguage(params.get("lang") ?? navigator.language, location.href);

const opened = opening(params);
const sessionID = await resolve(opened);
launch.hide();
if (params.get("session") !== sessionID) {
  // The address is now the session's, so a reload resumes it.
  history.replaceState(null, "", sessionAddress(sessionID, params));
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
// Tapped, so a dev server's debug recorder sees every frame; on any other
// server nobody listens.
const transport = new TappedTransport(new WebSocketTransport(liveURL(session, location.origin), () => recover()));
const openedAt = new Date().toISOString();
const playback: PlaybackEngine = new PlaybackEngine(
  transport,
  // One connection per seat: a scene of two characters has two.
  new SeatedVoice(session, (): readonly string[] | undefined => playback.state.topology?.seats),
  audio,
);
const typed = required<HTMLInputElement>("#typed");
const hold = required<HTMLButtonElement>("#hold");

const details = new Details(
  required<HTMLElement>("#details-title"),
  required<HTMLElement>("#details"),
  required<HTMLElement>("#usage"),
  session,
);
/** A block or wire picked anywhere opens its tab. */
const pick = (show: () => Promise<void>): void => {
  tabs.select("block");
  void show();
};
const guardrail = new Guardrail(
  required<HTMLElement>("#guardrail"),
  session,
  (name) => pick(() => details.show(name)),
);
const tabs = new InsightsTabs(
  required<HTMLElement>(".insights-details"),
  required<HTMLElement>("#insights-tabs"),
  (tab) => {
    // A folded lower pane opens when one of its tabs is picked.
    collapsibles.open("details");
    // The guardrail is read afresh whenever its tab opens: someone may have
    // changed it, here or in another tab.
    if (tab === "guardrail") void guardrail.show();
  },
);
const problems = new Problems(
  required<HTMLElement>("#problems"),
  (source) => pick(() => details.show(source)),
  (source) => !!playback.state.topology?.nodes.some((node) => node.name === source),
);

const view = new View({
  root: required<HTMLElement>("main"),
  title: required<HTMLElement>("#title"),
  props: required<HTMLElement>("#props"),
  graph: required<HTMLElement>("#graph"),
  background: required<HTMLElement>("#background"),
  scroll: required<HTMLElement>("#scroll"),
  feed: required<HTMLElement>("#feed"),
  preparing: required<HTMLElement>("#preparing"),
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
  onNodeClick: (name) => pick(() => details.show(name)),
  onBackgroundClick: () => {
    details.clear(playback.state);
    tabs.select("usage");
  },
  onEdgeClick: (from, to, kind) => pick(() => details.showEdge(from, to, kind)),
});

const insights = new Insights({
  toggle: required<HTMLButtonElement>("#insights-toggle"),
  label: required<HTMLElement>("#insights-label"),
  panel: required<HTMLElement>("#insights"),
  runsTitle: required<HTMLElement>("#runs-title"),
  runs: required<HTMLOListElement>("#runs"),
  close: required<HTMLButtonElement>("#details-close"),
}, {
  onRun: (name) => pick(() => details.show(name)),
  isBlock: (source) => !!playback.state.topology?.nodes.some((node) => node.name === source),
  onClose: () => details.clear(playback.state),
});

// The game's theme applies once the session says which. Trying the others is
// for developers: the select shows on a dev server only, as a game's theme
// comes from the game.
const themes = required<HTMLSelectElement>("#theme");
themes.append(...Object.keys(PRESETS).sort().map((name) => new Option(name, name)));

function useTheme(preset: string): void {
  themes.value = preset;
  view.setTheme(mergeTheme(fromGenerated({ preset })), preset);
}

useTheme("default");
themes.addEventListener("change", () => useTheme(themes.value));

mountSplit(required<HTMLElement>("#split"));
mountInsightsSplit(
  required<HTMLElement>("#insights-split"),
  required<HTMLElement>("#insights"),
  required<HTMLElement>(".insights-graph"),
  required<HTMLElement>(".insights-details"),
);
const collapsibles = new Collapsibles(required<HTMLElement>("#insights"), {
  graph: required<HTMLButtonElement>("#graph-fold"),
  details: required<HTMLButtonElement>("#details-fold"),
});

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
  insights.render(state);
  problems.render(state.journal);
  tabs.count("problems", Problems.count(state.journal));
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
  themes.hidden = !topology?.dev;
  // A developer can let Claude look into this session through the debug
  // relay; it records from here, before the live connection opens.
  let debugging: Debugging | null = null;
  if (topology?.dev) {
    debugging = startDebugging(
      playback,
      transport,
      { game: topology.title ?? topology.name, session: sessionID, started: openedAt },
      required<HTMLElement>(".insights-graph"),
    );
  }
  if (topology?.language) {
    // The session's language is fixed at creation, and wins over the page's.
    document.documentElement.lang = topology.language;
    await useLanguage(topology.language, location.href);
    talk.refresh();
    insights.relabel();
    tabs.relabel();
    collapsibles.relabel();
    debugging?.relabel();
    view.render(playback.state);
  }
  // A page that opens on the guardrail's tab reads it once the language is set.
  if (tabs.current === "guardrail") void guardrail.show();
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
 * Comes to a session: plays the one named, or creates one for a game or a
 * share, as the standalone server opens the page. On the platform the web
 * app has opened the session already; a refusal here is said plainly on the
 * launch panel, as the page has nothing else to offer.
 */
async function resolve(open: Opening): Promise<string> {
  switch (open.kind) {
    case "nothing":
      return launch.ask(text("launch.noGame"), []);
    case "resume": {
      const refused = await probe(open.id);
      if (!refused) return open.id;
      return launch.ask(text(refused.key), []);
    }
    case "new":
      return start(open.selector);
  }
}

/** Creates a session, or says why it cannot be had. */
async function start(selector: Selector): Promise<string> {
  const language = params.get("lang") ?? navigator.language;
  launch.wait(text("launch.starting"));
  const created = await create(client, selector, language, pending);
  if ("id" in created) return created.id;
  return launch.ask(text(created.notice.key), []);
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
