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
import { detailOf, type SessionRef } from "../client.js";
import { NoticeError, notice } from "../texts.js";
import type { Voice } from "../voice.js";

export class WebRTCVoice implements Voice {
  #peer: RTCPeerConnection | null = null;
  #microphone: MediaStream | null = null;
  #speaker: HTMLAudioElement | null = null;

  /**
   * @param session the session, which brokers the offer.
   * @param onRemoteTrack hands the character's audio to whatever plays it.
   */
  constructor(
    private readonly session: SessionRef,
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
      throw new NoticeError(notice("errors.noMicrophone"));
    }
    for (const track of this.#microphone.getAudioTracks()) {
      peer.addTrack(track, this.#microphone);
    }

    await peer.setLocalDescription(await peer.createOffer());
    await gathered(peer);

    const offer = peer.localDescription?.sdp;
    if (!offer) throw new Error("the browser produced no offer to send");

    let sdp: string;
    try {
      ({ data: { sdp } } = await this.session.client.postApiEngineSessionsIdConnect(this.session.id, { sdp: offer }));
    } catch (thrown) {
      throw new Error(`the engine refused the connection: ${await detailOf(thrown)}`);
    }
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
    try {
      await this.session.client.getApiEngineSessionsIdState(this.session.id);
    } catch (thrown) {
      if (!(thrown instanceof Response)) throw new NoticeError(notice("errors.serverUnreachable"));
      throw new NoticeError(notice("errors.voiceFailed", `the game server answered ${thrown.status}; this session may have ended`));
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
