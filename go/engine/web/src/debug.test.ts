import assert from "node:assert/strict";
import { test } from "node:test";
import { DebugRecorder, evaluate } from "./debug.js";
import { PlaybackEngine } from "./playback-engine.js";
import { TappedTransport, type Transport, type TransportHandlers } from "./transport.js";

/** A transport the test speaks through, standing in for the engine. */
class FakeTransport implements Transport {
  handlers: TransportHandlers | null = null;
  sent: (string | ArrayBuffer)[] = [];
  open(handlers: TransportHandlers): void {
    this.handlers = handlers;
    handlers.onOpen();
  }
  send(data: string | ArrayBuffer): void {
    this.sent.push(data);
  }
  close(): void {}
  receive(data: string | ArrayBuffer): void {
    this.handlers?.onMessage(data);
  }
}

/** An event on the session's stream, as the engine frames it. */
function event(stream: string, value: string): string {
  return JSON.stringify({ Stream: stream, Value: value });
}

/** A session on a fake engine, recorded from the start. */
function recordedSession(capacity?: number) {
  const engine = new FakeTransport();
  const tapped = new TappedTransport(engine);
  const playback = new PlaybackEngine(tapped);
  const recorder = new DebugRecorder(capacity);
  recorder.record(tapped, playback);
  playback.connect();
  return { engine, playback, recorder };
}

test("text frames are recorded in full both ways, audio frames by their size", () => {
  const { engine, playback, recorder } = recordedSession();

  engine.receive(event("text", "The bridge creaks."));
  playback.say("I cross it");
  engine.receive(new ArrayBuffer(960));
  playback.speak(new ArrayBuffer(480));

  const frames = recorder.entries().filter((e) => e.kind === "received" || e.kind === "sent");
  assert.deepEqual(
    frames.map(({ kind, text, bytes }) => ({ kind, text, bytes })),
    [
      { kind: "received", text: event("text", "The bridge creaks."), bytes: undefined },
      { kind: "sent", text: JSON.stringify({ text: "I cross it" }), bytes: undefined },
      { kind: "received", text: undefined, bytes: 960 },
      { kind: "sent", text: undefined, bytes: 480 },
    ],
  );
  assert.ok(frames.every((e) => !Number.isNaN(Date.parse(e.time))), "every entry says when");
});

test("phase changes and runs are recorded as they happen", () => {
  const { engine, recorder } = recordedSession();

  engine.receive(event("state", JSON.stringify({ node: "outline", phase: "working" })));
  engine.receive(event("state", JSON.stringify({ node: "outline", phase: "ready" })));

  const changes = recorder.entries().filter((e) => e.kind === "phase" || e.kind === "run");
  assert.deepEqual(
    changes.map(({ kind, node, from, to }) => ({ kind, node, from, to })),
    [
      { kind: "phase", node: "outline", from: undefined, to: "working" },
      { kind: "run", node: "outline", from: undefined, to: "started" },
      { kind: "phase", node: "outline", from: "working", to: "ready" },
      { kind: "run", node: "outline", from: undefined, to: "done" },
    ],
  );
});

test("notices and the connection's changes are recorded", () => {
  const { engine, recorder } = recordedSession();

  engine.receive(event("error", JSON.stringify({ code: "image_failed", text: "a bridge" })));
  engine.handlers?.onError();

  const said = recorder.entries().filter((e) => e.kind === "notice" || e.kind === "connection");
  assert.deepEqual(
    said.map(({ kind, to }) => ({ kind, to })),
    [
      { kind: "connection", to: "connecting" },
      { kind: "connection", to: "open" },
      { kind: "notice", to: "fault: errors.imageFailed" },
      { kind: "connection", to: "reconnecting" },
    ],
  );
});

test("an error the page reports is recorded", () => {
  const { recorder } = recordedSession();

  recorder.error("TypeError: x is undefined");

  assert.deepEqual(recorder.entries().at(-1)?.text, "TypeError: x is undefined");
  assert.equal(recorder.entries().at(-1)?.kind, "error");
});

test("only the latest entries are kept, oldest first", () => {
  const { engine, recorder } = recordedSession(3);

  for (let i = 1; i <= 5; i++) engine.receive(event("note", String(i)));

  assert.deepEqual(
    recorder.entries().map((e) => e.text),
    ["3", "4", "5"].map((n) => event("note", n)),
  );
});

test("a snapshot holds the playback engine's state and the buffer", () => {
  const { engine, playback, recorder } = recordedSession();
  engine.receive(event("text", "Fog."));

  const snapshot = recorder.snapshot(playback);

  assert.equal(snapshot.state.utterances[0]?.text, "Fog.");
  assert.deepEqual(snapshot.buffer, recorder.entries());
  assert.doesNotThrow(() => JSON.stringify(snapshot), "a snapshot travels as JSON");
});

test("an expression is evaluated with the session in scope, await included", async () => {
  const { engine, playback, recorder } = recordedSession();
  engine.receive(event("text", "Fog."));
  const scope = { playback, recorder, snapshot: () => recorder.snapshot(playback) };

  assert.deepEqual(await evaluate("playback.state.utterances.length", scope), { value: 1 });
  assert.deepEqual(await evaluate("await Promise.resolve(41 + 1)", scope), { value: 42 });
  assert.deepEqual(await evaluate("const n = snapshot().buffer.length; return n > 0", scope), { value: true });
  assert.deepEqual(await evaluate("undefined", scope), { value: null });
});

test("what an expression throws comes back as its error", async () => {
  const result = await evaluate("nope.at(1)", {});
  assert.match("error" in result ? result.error : "", /nope is not defined/);
});

test("a result that is no JSON comes back readable", async () => {
  const result = await evaluate("const a = { buf: new ArrayBuffer(8), m: new Map([[1, 2]]) }; a.self = a; return a", {});
  assert.deepEqual(result, { value: { buf: "ArrayBuffer(8)", m: [[1, 2]], self: "[circular]" } });
});
