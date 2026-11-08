/**
 * The player's own audio connection, over WebRTC.
 *
 * Audio runs between this browser and the model. The engine is not on that
 * path: it creates the session, holds the API key, brokers the offer below, and
 * then watches the conversation over a control connection of its own. So the
 * page never sees a spendable key, and the engine never relays a megabyte a
 * minute of speech for every player.
 *
 * It lives under ui/ because it needs a microphone and a speaker, and the core
 * has neither — the same reason the graph library lives here.
 */
import type { Voice } from "../voice.js";

export class WebRTCVoice implements Voice {
  #peer: RTCPeerConnection | null = null;
  #microphone: MediaStream | null = null;
  #speaker: HTMLAudioElement | null = null;

  /**
   * @param base the session's URL, which is where the offer is brokered.
   * @param onRemoteTrack hands the character's audio to whatever plays it.
   */
  constructor(
    private readonly base: URL,
    private readonly onRemoteTrack?: (stream: MediaStream) => void,
  ) {}

  async connect(): Promise<void> {
    // Checked before anything is announced or asked for. A page that says it is
    // opening a microphone while the engine is unreachable blames the wrong
    // thing, and asks somebody for permission it has no use for.
    await this.#reachable();

    const peer = new RTCPeerConnection();
    this.#peer = peer;

    // The data channel is created before the offer, because it has to be
    // described in the SDP the engine is about to exchange.
    peer.createDataChannel("oai-events");

    peer.ontrack = (event) => {
      const [stream] = event.streams;
      if (!stream) return;
      this.#play(stream);
      this.onRemoteTrack?.(stream);
    };

    try {
      this.#microphone = await navigator.mediaDevices.getUserMedia({
        audio: { channelCount: 1, echoCancellation: true, noiseSuppression: true },
      });
    } catch {
      throw new Error("No microphone. Check the browser's permission for this page.");
    }
    for (const track of this.#microphone.getAudioTracks()) {
      peer.addTrack(track, this.#microphone);
    }

    await peer.setLocalDescription(await peer.createOffer());
    await gathered(peer);

    const offer = peer.localDescription?.sdp;
    if (!offer) throw new Error("the browser produced no offer to send");

    const response = await fetch(new URL("connect", this.base), {
      method: "POST",
      headers: { "Content-Type": "application/json" },
      body: JSON.stringify({ sdp: offer }),
    });
    if (!response.ok) {
      throw new Error(`the engine refused the connection: ${response.status} ${await response.text()}`);
    }

    const { sdp } = (await response.json()) as { sdp: string };
    // The answer took a round trip, and this connection may have been let go
    // while it was away — a pause closes the peer, and a later attempt replaces
    // it. Answering a peer that is no longer the one in use fails, and the
    // failure would be reported as a voice that could not be opened.
    if (this.#peer !== peer || peer.signalingState === "closed") {
      throw new Error("the conversation was let go while it was connecting");
    }
    await peer.setRemoteDescription({ type: "answer", sdp });
  }

  /** Confirms the engine is answering before the rest of this is attempted. */
  async #reachable(): Promise<void> {
    let response: Response;
    try {
      response = await fetch(new URL("state", this.base));
    } catch {
      throw new Error("Cannot reach the game server.");
    }
    if (!response.ok) {
      throw new Error(`The game server answered ${response.status}; this session may have ended.`);
    }
  }

  setSending(on: boolean): void {
    for (const track of this.#microphone?.getAudioTracks() ?? []) {
      track.enabled = on;
    }
  }

  close(): void {
    for (const track of this.#microphone?.getAudioTracks() ?? []) track.stop();
    this.#microphone = null;
    this.#peer?.close();
    this.#peer = null;
    if (this.#speaker) {
      this.#speaker.srcObject = null;
      this.#speaker.remove();
      this.#speaker = null;
    }
  }

  #play(stream: MediaStream): void {
    if (!this.#speaker) {
      // An element rather than the Web Audio API: WebRTC already delivers a
      // playable stream with its jitter buffer attached, and scheduling it by
      // hand would be re-solving what the transport did.
      this.#speaker = document.createElement("audio");
      this.#speaker.autoplay = true;
      document.body.append(this.#speaker);
    }
    this.#speaker.srcObject = stream;
  }
}

/**
 * Waits for ICE gathering to finish, because the offer must carry its
 * candidates: the engine brokers one exchange rather than trickling.
 */
function gathered(peer: RTCPeerConnection): Promise<void> {
  if (peer.iceGatheringState === "complete") return Promise.resolve();

  return new Promise((resolve) => {
    const done = (): void => {
      if (peer.iceGatheringState !== "complete") return;
      peer.removeEventListener("icegatheringstatechange", done);
      clearTimeout(timer);
      resolve();
    };
    // Gathering can stall on an unreachable STUN server, and a slightly
    // incomplete offer connects more often than one that never gets sent.
    const timer = setTimeout(() => {
      peer.removeEventListener("icegatheringstatechange", done);
      resolve();
    }, 3_000);
    peer.addEventListener("icegatheringstatechange", done);
  });
}
