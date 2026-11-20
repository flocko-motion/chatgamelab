/**
 * Microphone capture and playback, at the 24 kHz mono PCM16 the realtime API
 * speaks in both directions.
 *
 * Playback schedules each chunk after the previous one rather than starting it
 * now, so a stream of deltas plays as continuous speech instead of overlapping
 * itself.
 */

const SAMPLE_RATE = 24_000;

export class AudioIO {
  #context: AudioContext | null = null;
  #stream: MediaStream | null = null;
  #capture: AudioWorkletNode | null = null;
  #playAt = 0;

  constructor(private readonly workletURL: string) {}

  /** Lazily created: a context made before a user gesture starts suspended. */
  async #ready(): Promise<AudioContext> {
    if (!this.#context) {
      this.#context = new AudioContext({ sampleRate: SAMPLE_RATE });
      await this.#context.audioWorklet.addModule(this.workletURL);
    }
    if (this.#context.state === "suspended") await this.#context.resume();
    return this.#context;
  }

  async startCapture(onFrame: (pcm: ArrayBuffer) => void): Promise<void> {
    const context = await this.#ready();
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

  async play(pcm: ArrayBuffer): Promise<void> {
    const context = await this.#ready();
    const samples = new Int16Array(pcm);
    if (samples.length === 0) return;

    const frame = context.createBuffer(1, samples.length, SAMPLE_RATE);
    const channel = frame.getChannelData(0);
    for (let i = 0; i < samples.length; i++) channel[i] = (samples[i] ?? 0) / 32768;

    const source = context.createBufferSource();
    source.buffer = frame;
    source.connect(context.destination);

    this.#playAt = Math.max(this.#playAt, context.currentTime);
    source.start(this.#playAt);
    this.#playAt += frame.duration;
  }

  /** Drops queued playback, for when a turn is cut short. */
  resetPlayback(): void {
    this.#playAt = 0;
  }
}
