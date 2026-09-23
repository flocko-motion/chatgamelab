/**
 * The core talks to a transport, not to a WebSocket, so a test can drive a full
 * session with no server and no browser.
 */
export type TransportMessage = string | ArrayBuffer;

export interface TransportHandlers {
  onOpen(): void;
  onClose(): void;
  onError(): void;
  onMessage(data: TransportMessage): void;
}

export interface Transport {
  open(handlers: TransportHandlers): void;
  /** Binary frames are audio; text frames are everything else. */
  send(data: ArrayBuffer | string): void;
  close(): void;
}

/**
 * The real transport, which puts itself back together.
 *
 * A session outlives its socket. A laptop sleeps, a network hiccups, a portal
 * is redeployed — none of which ends the game, and on a voice genre none of
 * which even interrupts the conversation, since that audio runs between the
 * browser and the model. A socket that stayed down would leave a playable game
 * looking dead.
 *
 * So a drop is retried, with a backoff that gives up nothing but haste: quick
 * enough that a blink is invisible, slow enough that an unreachable server is
 * not hammered. `onClose` is reported only when the retrying stops, so a caller
 * hears "the connection is gone" once rather than once per attempt.
 *
 * WebSocket is a global in browsers and in Node 22+, so the same implementation
 * serves the page and a headless run.
 */
export class WebSocketTransport implements Transport {
  #socket: WebSocket | null = null;
  #handlers: TransportHandlers | null = null;
  #retry: ReturnType<typeof setTimeout> | null = null;
  #attempt = 0;
  /** Set by close(), so a deliberate hang-up is not treated as a fault. */
  #done = false;

  /**
   * @param url the session's stream.
   * @param onReopen called when a dropped connection comes back, so whoever is
   *   listening can catch up on what it missed while it was away.
   */
  constructor(
    private readonly url: URL,
    private readonly onReopen?: () => void,
  ) {}

  open(handlers: TransportHandlers): void {
    this.#handlers = handlers;
    this.#done = false;
    this.#connect(false);
  }

  #connect(reopening: boolean): void {
    if (this.#done) return;

    const socket = new WebSocket(this.url);
    socket.binaryType = "arraybuffer";
    this.#socket = socket;

    socket.onopen = () => {
      this.#attempt = 0;
      this.#handlers?.onOpen();
      // Announced after the open, so anything catching up is doing so against
      // a connection that is already live.
      if (reopening) this.onReopen?.();
    };
    socket.onmessage = (message: MessageEvent<TransportMessage>) =>
      this.#handlers?.onMessage(message.data);
    // A failed socket also closes, so the retry is driven from one place.
    socket.onerror = () => {};
    socket.onclose = () => {
      this.#socket = null;
      if (this.#done) {
        this.#handlers?.onClose();
        return;
      }
      this.#schedule();
    };
  }

  /**
   * Waits before trying again, doubling up to a ceiling. The wait is jittered
   * because a restarted server is reconnected to by every player at once, and
   * identical backoffs would arrive as one thundering herd.
   */
  #schedule(): void {
    const ceiling = 10_000;
    const base = Math.min(500 * 2 ** this.#attempt, ceiling);
    this.#attempt++;

    this.#handlers?.onError();
    this.#retry = setTimeout(() => this.#connect(true), base * (0.5 + Math.random() / 2));
  }

  send(data: ArrayBuffer | string): void {
    if (this.#socket?.readyState === WebSocket.OPEN) this.#socket.send(data);
  }

  close(): void {
    this.#done = true;
    if (this.#retry) {
      clearTimeout(this.#retry);
      this.#retry = null;
    }
    const socket = this.#socket;
    this.#socket = null;
    if (socket) socket.close();
    else this.#handlers?.onClose();
  }
}
