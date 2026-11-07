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
 * A session outlives its socket, so a drop is retried with backoff; `onClose`
 * fires once, when retrying stops.
 */
export class WebSocketTransport implements Transport {
  #socket: WebSocket | null = null;
  #handlers: TransportHandlers | null = null;
  #retry: ReturnType<typeof setTimeout> | null = null;
  #attempt = 0;
  /** Set by close(), so a deliberate hang-up is not treated as a fault. */
  #done = false;

  /** @param onReopen called when a dropped connection returns, so a listener can catch up. */
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
      // After the open, so catching up runs against a live connection.
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

  /** Jittered, so a restarted server is not reconnected to by every player at once. */
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
