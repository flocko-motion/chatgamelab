import assert from "node:assert/strict";
import { mock, test } from "node:test";
import { SILENCE_LIMIT, WebSocketTransport } from "./transport.js";

/** A socket the test opens, feeds and watches being closed. */
class FakeSocket {
  static OPEN = 1;
  static made: FakeSocket[] = [];
  readyState = 0;
  binaryType = "blob";
  closed = false;
  onopen: (() => void) | null = null;
  onmessage: ((m: { data: string }) => void) | null = null;
  onerror: (() => void) | null = null;
  onclose: (() => void) | null = null;
  constructor(readonly url: URL) {
    FakeSocket.made.push(this);
  }
  open(): void {
    this.readyState = FakeSocket.OPEN;
    this.onopen?.();
  }
  hear(data: string): void {
    this.onmessage?.({ data });
  }
  send(): void {}
  close(): void {
    this.closed = true;
    this.readyState = 3;
    this.onclose?.();
  }
}

function withFakes(run: () => void): void {
  const real = globalThis.WebSocket;
  (globalThis as { WebSocket: unknown }).WebSocket = FakeSocket;
  FakeSocket.made = [];
  mock.timers.enable({ apis: ["setInterval", "setTimeout", "Date"] });
  try {
    run();
  } finally {
    mock.timers.reset();
    (globalThis as { WebSocket: unknown }).WebSocket = real;
  }
}

const handlers = { onOpen() {}, onClose() {}, onError() {}, onMessage() {} };

test("a socket the engine has gone silent on is closed, reopened, and caught up", () => {
  withFakes(() => {
    let reopened = 0;
    const transport = new WebSocketTransport(new URL("ws://engine/live"), () => reopened++);
    transport.open(handlers);
    const first = FakeSocket.made[0]!;
    first.open();

    mock.timers.tick(SILENCE_LIMIT + 5_000);
    assert.equal(first.closed, true, "the silent socket was not closed");

    mock.timers.tick(10_000);
    const second = FakeSocket.made[1];
    assert.ok(second, "no new socket was opened");
    second.open();
    assert.equal(reopened, 1, "the listener was not asked to catch up");
    transport.close();
  });
});

test("a socket that hears heartbeats stays open", () => {
  withFakes(() => {
    const transport = new WebSocketTransport(new URL("ws://engine/live"));
    transport.open(handlers);
    const socket = FakeSocket.made[0]!;
    socket.open();
    for (let i = 0; i < 6; i++) {
      mock.timers.tick(20_000);
      socket.hear(JSON.stringify({ Stream: "heartbeat", Value: "" }));
    }
    assert.equal(socket.closed, false, "a socket hearing its heartbeat was closed");
    assert.equal(FakeSocket.made.length, 1);
    transport.close();
  });
});
