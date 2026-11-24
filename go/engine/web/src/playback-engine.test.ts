import assert from "node:assert/strict";
import { test } from "node:test";
import { PlaybackEngine } from "./playback-engine.js";
import type { Speaker } from "./speaker.js";
import type { Transport, TransportHandlers } from "./transport.js";

/** A transport the test speaks through, standing in for the engine. */
class FakeTransport implements Transport {
  handlers: TransportHandlers | null = null;
  open(handlers: TransportHandlers): void {
    this.handlers = handlers;
    handlers.onOpen();
  }
  send(): void {}
  close(): void {}
  receive(data: string | ArrayBuffer): void {
    this.handlers?.onMessage(data);
  }
}

/** A speaker that records what it was asked to do. */
class RecordingSpeaker implements Speaker {
  calls: string[] = [];
  play(pcm: ArrayBuffer): void {
    this.calls.push(`play ${pcm.byteLength}`);
  }
  discard(): void {
    this.calls.push("discard");
  }
}

function frame(bytes: number): ArrayBuffer {
  return new ArrayBuffer(bytes);
}

test("each audio frame is played as it arrives, and counted", () => {
  const transport = new FakeTransport();
  const speaker = new RecordingSpeaker();
  const playback = new PlaybackEngine(transport, undefined, speaker);
  playback.connect();

  transport.receive(frame(960));
  transport.receive(frame(480));

  assert.deepEqual(speaker.calls, ["play 960", "play 480"]);
  assert.equal(playback.state.audioChunks, 2);
});

test("an audio discard stops the speech and drops what is queued", () => {
  const transport = new FakeTransport();
  const speaker = new RecordingSpeaker();
  const playback = new PlaybackEngine(transport, undefined, speaker);
  playback.connect();

  transport.receive(frame(960));
  transport.receive(frame(0));
  transport.receive(frame(480));

  // The retried reply speaks from its start.
  assert.deepEqual(speaker.calls, ["play 960", "discard", "play 480"]);
  assert.equal(playback.state.audioChunks, 2, "a discard is no audio");
  assert.equal(playback.state.audioDiscards, 1);
});

test("a headless run counts frames and discards with no speaker", () => {
  const transport = new FakeTransport();
  const playback = new PlaybackEngine(transport);
  playback.connect();

  transport.receive(frame(960));
  transport.receive(frame(0));

  assert.equal(playback.state.audioChunks, 1);
  assert.equal(playback.state.audioDiscards, 1);
});

/** An event on the session's stream, as the engine frames it. */
function event(stream: string, value: string): string {
  return JSON.stringify({ Stream: stream, Value: value });
}

test("a typed line is drawn when the engine sends it back on said", () => {
  const transport = new FakeTransport();
  const playback = new PlaybackEngine(transport);
  playback.connect();

  playback.say("I cross the bridge");
  assert.equal(playback.state.utterances.length, 0, "the player's line comes from the engine");

  transport.receive(event("said", "I cross the bridge"));
  assert.deepEqual(
    playback.state.utterances.map((u) => [u.speaker, u.text]),
    [["player", "I cross the bridge"]],
  );
  assert.equal(playback.state.hearing, false);
});

test("a spoken line waits for what was heard, which is given back if its turn fails", () => {
  const transport = new FakeTransport();
  const playback = new PlaybackEngine(transport);
  playback.connect();
  transport.receive(event("text", "The troll looks up."));

  playback.speak(new ArrayBuffer(48_000));
  assert.equal(playback.state.hearing, true);

  transport.receive(event("said", "ich überquere die Brücke"));
  assert.equal(playback.state.hearing, false);
  assert.deepEqual(
    playback.state.utterances.map((u) => [u.speaker, u.text, u.complete]),
    [["npc", "The troll looks up.", true], ["player", "ich überquere die Brücke", true]],
    "the player's line ends the reply before it",
  );

  transport.receive(event("error", JSON.stringify({ code: "turn_failed", text: "The player crosses." })));
  assert.equal(playback.state.redo, "ich überquere die Brücke", "what was heard goes back into the box");
});

test("a clip of which nothing came through ends its wait and says so", () => {
  const transport = new FakeTransport();
  const playback = new PlaybackEngine(transport);
  playback.connect();

  playback.speak(new ArrayBuffer(48_000));
  transport.receive(event("error", JSON.stringify({ code: "voice_failed", text: "" })));

  assert.equal(playback.state.hearing, false);
  assert.equal(playback.state.fault?.key, "errors.notHeard");
  assert.equal(playback.state.utterances.length, 0);
});

test("a replayed voice failure is long past", () => {
  const transport = new FakeTransport();
  const playback = new PlaybackEngine(transport);
  playback.connect();

  transport.receive(event("error", JSON.stringify({ code: "voice_failed", text: "" })));
  assert.equal(playback.state.fault, null);
});

test("each block that runs is one entry in what happened, finished once it is ready again", () => {
  const transport = new FakeTransport();
  const playback = new PlaybackEngine(transport);
  playback.connect();
  const block = (node: string, phase: string) => transport.receive(event("state", JSON.stringify({ node, phase })));

  block("expand", "working");
  block("image", "working");
  block("expand", "ready");
  block("expand", "working");

  assert.deepEqual(
    playback.state.runs.map((r) => [r.node, r.done]),
    [
      ["expand", true],
      ["image", false],
      ["expand", false],
    ],
  );
});

test("a block reported ready without having run adds nothing", () => {
  const transport = new FakeTransport();
  const playback = new PlaybackEngine(transport);
  playback.connect();

  transport.receive(event("state", JSON.stringify({ node: "gate", phase: "ready" })));

  assert.deepEqual(playback.state.runs, []);
});

test("journal events gather in the journal, not among the notes", () => {
  const transport = new FakeTransport();
  const playback = new PlaybackEngine(transport, undefined, new RecordingSpeaker());
  playback.connect();

  const entry = {
    time: "2026-11-23T21:44:49.402Z",
    source: "live-session",
    level: "warning",
    kind: "closed",
    detail: "the provider closed the conversation: remote_hangup",
  };
  transport.receive(JSON.stringify({ Stream: "journal", Value: JSON.stringify(entry) }));

  assert.deepEqual(playback.state.journal, [entry]);
  assert.deepEqual(playback.state.notes, []);
});

/** A voice that records what it was asked to do. */
class RecordingVoice {
  calls: string[] = [];
  async connect(): Promise<void> {
    this.calls.push("connect");
  }
  close(): void {
    this.calls.push("close");
  }
  setSending(): void {}
}

const invitation = (reason = "") => JSON.stringify({ Stream: "connect", Value: reason });

test("an invitation after the provider dropped the conversation connects again by itself", async () => {
  const transport = new FakeTransport();
  const voice = new RecordingVoice();
  const playback = new PlaybackEngine(transport, voice, new RecordingSpeaker());
  playback.connect();

  transport.receive(invitation());
  await playback.until((s) => s.voice === "live", 1000);
  // The provider hung up; the engine invites again, without waiting for a press.
  transport.receive(invitation());
  await playback.until(() => voice.calls.length === 3, 1000);

  assert.deepEqual(voice.calls, ["connect", "close", "connect"], "the dead connection is replaced");
  assert.equal(playback.state.paused, false);
});

test("an invitation that waits for the press does not connect", () => {
  const transport = new FakeTransport();
  const voice = new RecordingVoice();
  const playback = new PlaybackEngine(transport, voice, new RecordingSpeaker());
  playback.connect();

  transport.receive(invitation("press"));

  assert.deepEqual(voice.calls, []);
  assert.equal(playback.state.paused, true, "the player is offered the press");
  assert.equal(playback.state.invited, true);
});

test("a conversation let go for silence shows paused, and the press picks it up", async () => {
  const transport = new FakeTransport();
  const voice = new RecordingVoice();
  const playback = new PlaybackEngine(transport, voice, new RecordingSpeaker());
  playback.connect();

  transport.receive(invitation());
  await playback.until((s) => s.voice === "live", 1000);
  // The engine let the conversation go: nobody had spoken for a while.
  transport.receive(JSON.stringify({ Stream: "pause", Value: "" }));

  assert.equal(playback.state.paused, true, "the game shows paused");
  assert.equal(playback.state.voice, "idle");
  assert.deepEqual(voice.calls, ["connect", "close"], "the audio is let go with it");

  // It waits for the press, then carries on.
  transport.receive(invitation("press"));
  assert.deepEqual(voice.calls, ["connect", "close"], "nothing connects before the press");
  playback.resume();
  await playback.until((s) => s.voice === "live", 1000);
  assert.equal(playback.state.paused, false);
  assert.deepEqual(voice.calls, ["connect", "close", "connect"]);
});

test("a game that is over lets the conversation go and offers nothing to press", async () => {
  const { audioState, acceptsTypedInput, holdsToTalk } = await import("./state.js");
  const transport = new FakeTransport();
  const voice = new RecordingVoice();
  const playback = new PlaybackEngine(transport, voice, new RecordingSpeaker());
  playback.connect();

  transport.receive(invitation());
  await playback.until((s) => s.voice === "live", 1000);
  transport.receive(JSON.stringify({ Stream: "over", Value: "content_policy" }));

  assert.equal(playback.state.over, "content_policy");
  assert.deepEqual(voice.calls, ["connect", "close"], "the audio is let go for good");
  assert.equal(audioState(playback.state), null, "no pause or resume control");
  assert.equal(acceptsTypedInput(playback.state), false);
  assert.equal(holdsToTalk(playback.state), false);

  // A late invitation does not bring the controls back.
  playback.resume();
  assert.deepEqual(voice.calls, ["connect", "close"]);
});

test("an end this page does not know is told as an end without a reason", async () => {
  const { overReason } = await import("./state.js");
  assert.equal(overReason("content_policy"), "content_policy");
  assert.equal(overReason("something_new"), "unknown");
});

test("a toggle is worth offering only where it switches a block of the wiring", async () => {
  const { switches } = await import("./state.js");
  const playback = new PlaybackEngine(new FakeTransport(), undefined, new RecordingSpeaker());
  assert.equal(switches(playback.state, "images"), false, "no wiring yet, nothing to switch");
  playback.state.topology = { switches: ["images"] } as unknown as typeof playback.state.topology;
  assert.equal(switches(playback.state, "images"), true);
  assert.equal(switches(playback.state, "audio-out"), false);
});
