/**
 * Captures microphone input as PCM16 frames on the audio thread.
 *
 * A worklet rather than a ScriptProcessorNode: the deprecated node ran its
 * callback on the main thread, where a slow render could drop audio.
 */

declare const sampleRate: number;
declare abstract class AudioWorkletProcessor {
  readonly port: MessagePort;
  abstract process(inputs: Float32Array[][]): boolean;
}
declare function registerProcessor(
  name: string,
  processor: new () => AudioWorkletProcessor,
): void;

class MicCapture extends AudioWorkletProcessor {
  override process(inputs: Float32Array[][]): boolean {
    const channel = inputs[0]?.[0];
    if (!channel || channel.length === 0) return true;

    const pcm = new Int16Array(channel.length);
    for (let i = 0; i < channel.length; i++) {
      const sample = Math.max(-1, Math.min(1, channel[i] ?? 0));
      pcm[i] = sample < 0 ? sample * 32768 : sample * 32767;
    }
    this.port.postMessage(pcm.buffer, [pcm.buffer]);
    return true;
  }
}

registerProcessor("mic-capture", MicCapture);
export {};
