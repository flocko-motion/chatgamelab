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
