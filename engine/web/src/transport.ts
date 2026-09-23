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
  send(data: ArrayBuffer): void;
  close(): void;
}

/**
 * The real transport. WebSocket is a global in browsers and in Node 22+, so the
 * same implementation serves the page and a headless run.
 */
export class WebSocketTransport implements Transport {
  #socket: WebSocket | null = null;

  constructor(private readonly url: URL) {}

  open(handlers: TransportHandlers): void {
    const socket = new WebSocket(this.url);
    socket.binaryType = "arraybuffer";
    this.#socket = socket;

    socket.onopen = () => handlers.onOpen();
    socket.onclose = () => handlers.onClose();
    socket.onerror = () => handlers.onError();
    socket.onmessage = (message: MessageEvent<TransportMessage>) =>
      handlers.onMessage(message.data);
  }

  send(data: ArrayBuffer): void {
    if (this.#socket?.readyState === WebSocket.OPEN) this.#socket.send(data);
  }

  close(): void {
    this.#socket?.close();
    this.#socket = null;
  }
}
