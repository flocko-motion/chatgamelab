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
 * How long the page waits without hearing from the engine before it takes the
 * socket for dead. The engine sends a heartbeat every 20 s, so this is two
 * missed beats and a margin.
 */
export const SILENCE_LIMIT = 45_000;

/** How long without a frame makes a page coming back check its socket at once. */
const STALE_AFTER = 25_000;

/**
 * A session outlives its socket, so a drop is retried with backoff; `onClose`
 * fires once, when retrying stops. A socket the engine has gone quiet on, as
 * after a laptop slept or a network changed, is closed and opened again, and
 * whoever listens catches up through `onReopen`.
 */
export class WebSocketTransport implements Transport {
  #socket: WebSocket | null = null;
  #handlers: TransportHandlers | null = null;
  #retry: ReturnType<typeof setTimeout> | null = null;
  #attempt = 0;
  /** Set by close(), so a deliberate hang-up is not treated as a fault. */
  #done = false;
  /** When the engine was last heard from, on the socket now open. */
  #heard = 0;
  #watchdog: ReturnType<typeof setInterval> | null = null;

  /** @param onReopen called when a dropped connection returns, so a listener can catch up. */
  constructor(
    private readonly url: URL,
    private readonly onReopen?: () => void,
  ) {}

  open(handlers: TransportHandlers): void {
    this.#handlers = handlers;
    this.#done = false;
    this.#connect(false);
    this.#watch();
  }

  /** Closes a socket gone silent, so it reopens; a page coming back checks at once. */
  #watch(): void {
    this.#watchdog = setInterval(() => this.#check(SILENCE_LIMIT), 5_000);
    if (typeof document !== "undefined") {
      document.addEventListener("visibilitychange", this.#onReturn);
    }
    if (typeof window !== "undefined") window.addEventListener("online", this.#onReturn);
  }

  #onReturn = (): void => {
    if (typeof document === "undefined" || document.visibilityState === "visible") this.#check(STALE_AFTER);
  };

  #check(limit: number): void {
    const socket = this.#socket;
    if (!socket || socket.readyState !== WebSocket.OPEN) return;
    // Closing hands over to the retry, which reopens and catches up.
    if (Date.now() - this.#heard > limit) socket.close();
  }

  #connect(reopening: boolean): void {
    if (this.#done) return;

    const socket = new WebSocket(this.url);
    socket.binaryType = "arraybuffer";
    this.#socket = socket;

    socket.onopen = () => {
      this.#attempt = 0;
      this.#heard = Date.now();
      this.#handlers?.onOpen();
      // After the open, so catching up runs against a live connection.
      if (reopening) this.onReopen?.();
    };
    socket.onmessage = (message: MessageEvent<TransportMessage>) => {
      this.#heard = Date.now();
      this.#handlers?.onMessage(message.data);
    };
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
    if (this.#watchdog) {
      clearInterval(this.#watchdog);
      this.#watchdog = null;
    }
    if (typeof document !== "undefined") document.removeEventListener("visibilitychange", this.#onReturn);
    if (typeof window !== "undefined") window.removeEventListener("online", this.#onReturn);
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

/** Which way a frame went: from the engine, or to it. */
export type Direction = "received" | "sent";

export type FrameListener = (direction: Direction, data: TransportMessage) => void;

/**
 * A transport that shows every frame, both ways, to whoever listens, so a
 * developer's recorder can be attached once the page knows it runs on a dev
 * server. With nobody listening it only passes the frames on.
 */
export class TappedTransport implements Transport {
  #listener: FrameListener | null = null;

  constructor(private readonly inner: Transport) {}

  /** Shows every frame from now on to one listener. */
  listen(listener: FrameListener): void {
    this.#listener = listener;
  }

  open(handlers: TransportHandlers): void {
    this.inner.open({
      ...handlers,
      onMessage: (data) => {
        this.#listener?.("received", data);
        handlers.onMessage(data);
      },
    });
  }

  send(data: ArrayBuffer | string): void {
    this.#listener?.("sent", data);
    this.inner.send(data);
  }

  close(): void {
    this.inner.close();
  }
}
