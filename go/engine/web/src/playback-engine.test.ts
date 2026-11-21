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
