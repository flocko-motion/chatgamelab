/**
 * Microphone capture and playback, at the 24 kHz mono PCM16 the realtime API
 * speaks in both directions.
 *
 * Playback schedules each chunk after the previous one rather than starting it
 * now, so a stream of deltas plays as continuous speech instead of overlapping
 * itself. It is the page's speaker: the playback-engine hands it each frame of
 * a spoken reply as it arrives.
 */
import type { Speaker } from "../speaker.js";

const SAMPLE_RATE = 24_000;

export class AudioIO implements Speaker {
  #context: AudioContext | null = null;
  #worklet: Promise<void> | null = null;
  #stream: MediaStream | null = null;
  #capture: AudioWorkletNode | null = null;
  #playAt = 0;
  /** Frames wait their turn here, so they are scheduled in the order they came. */
  #queue: Promise<void> = Promise.resolve();
  /** Sources scheduled and not yet ended, which a discard stops. */
  #sources = new Set<AudioBufferSourceNode>();
  /** Bumped by a discard, so frames queued before it are dropped. */
  #generation = 0;

  /** @param workletURL the capture worklet, loaded only when capture starts. */
  constructor(private readonly workletURL: string) {}

  /** Lazily created: a context made before a user gesture starts suspended. */
  async #ready(): Promise<AudioContext> {
    if (!this.#context) this.#context = new AudioContext({ sampleRate: SAMPLE_RATE });
    if (this.#context.state === "suspended") await this.#context.resume();
    return this.#context;
  }

  async startCapture(onFrame: (pcm: ArrayBuffer) => void): Promise<void> {
    const context = await this.#ready();
    this.#worklet ??= context.audioWorklet.addModule(this.workletURL);
    await this.#worklet;
    this.#stream = await navigator.mediaDevices.getUserMedia({
      audio: { channelCount: 1, sampleRate: SAMPLE_RATE, echoCancellation: true },
    });

    const source = context.createMediaStreamSource(this.#stream);
    this.#capture = new AudioWorkletNode(context, "mic-capture");
    this.#capture.port.onmessage = (event: MessageEvent<ArrayBuffer>) => onFrame(event.data);
    source.connect(this.#capture);
    // Not connected to the destination: capture must not echo into playback.
  }

  stopCapture(): void {
    this.#capture?.port.close();
    this.#capture?.disconnect();
    this.#capture = null;
    this.#stream?.getTracks().forEach((track) => track.stop());
    this.#stream = null;
  }

  play(pcm: ArrayBuffer): void {
    const generation = this.#generation;
    this.#queue = this.#queue
      .then(() => (generation === this.#generation ? this.#schedule(pcm, generation) : undefined))
      // A frame that cannot play costs only that frame.
      .catch(() => {});
  }

  async #schedule(pcm: ArrayBuffer, generation: number): Promise<void> {
    const context = await this.#ready();
    if (generation !== this.#generation) return;
    const samples = new Int16Array(pcm);
    if (samples.length === 0) return;

    const frame = context.createBuffer(1, samples.length, SAMPLE_RATE);
    const channel = frame.getChannelData(0);
    for (let i = 0; i < samples.length; i++) channel[i] = (samples[i] ?? 0) / 32768;

    const source = context.createBufferSource();
    source.buffer = frame;
    source.connect(context.destination);
    this.#sources.add(source);
    source.onended = () => this.#sources.delete(source);

    this.#playAt = Math.max(this.#playAt, context.currentTime);
    source.start(this.#playAt);
    this.#playAt += frame.duration;
  }

  /** Stops what is playing and drops what is queued, for a reply made again. */
  discard(): void {
    this.#generation++;
    for (const source of this.#sources) source.stop();
    this.#sources.clear();
    this.#playAt = 0;
  }
}
