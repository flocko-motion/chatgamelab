import { useState, useCallback, useRef, useEffect } from "react";
import { uiLogger } from "@/config/logger";

// ── Constants ────────────────────────────────────────────────────────────

const MAX_RECORDING_SECONDS = 10;
const PREFERRED_MIME_TYPES = [
  "audio/webm;codecs=opus",
  "audio/webm",
  "audio/ogg;codecs=opus",
  "audio/ogg",
  "audio/mp4",
];

// ── Types ────────────────────────────────────────────────────────────────

export type RecordingState = "idle" | "requesting" | "recording" | "processing";

export interface AudioRecorderResult {
  /** Current recording state */
  state: RecordingState;
  /** Elapsed recording time in seconds (updates every ~100ms while recording) */
  elapsed: number;
  /** Maximum recording duration in seconds */
  maxDuration: number;
  /** Whether the browser supports audio recording */
  isSupported: boolean;
  /** Start recording (requests mic permission if needed) */
  startRecording: () => void;
  /** Stop recording and produce the audio blob */
  stopRecording: () => void;
  /** Cancel recording without producing output */
  cancelRecording: () => void;
  /** Error message if something went wrong */
  error: string | null;
}

// ── Helpers ──────────────────────────────────────────────────────────────

function getSupportedMimeType(): string | undefined {
  if (typeof MediaRecorder === "undefined") return undefined;
  return PREFERRED_MIME_TYPES.find((type) => MediaRecorder.isTypeSupported(type));
}

function blobToBase64(blob: Blob): Promise<string> {
  return new Promise((resolve, reject) => {
    const reader = new FileReader();
    reader.onloadend = () => {
      const dataUrl = reader.result as string;
      // Strip the data URL prefix (e.g. "data:audio/webm;base64,")
      const base64 = dataUrl.split(",")[1];
      resolve(base64);
    };
    reader.onerror = reject;
    reader.readAsDataURL(blob);
  });
}

// ── Hook ─────────────────────────────────────────────────────────────────

/**
 * Hook for recording audio from the user's microphone.
 *
 * Uses MediaRecorder API with a 15-second limit.
 * Returns a base64-encoded audio blob via the onComplete callback.
 *
 * @param onComplete Called with { base64, mimeType } when recording finishes successfully.
 */
export function useAudioRecorder(
  onComplete: (audio: { base64: string; mimeType: string }) => void,
): AudioRecorderResult {
  const [state, setState] = useState<RecordingState>("idle");
  const [elapsed, setElapsed] = useState(0);
  const [error, setError] = useState<string | null>(null);

  const mediaRecorderRef = useRef<MediaRecorder | null>(null);
  const streamRef = useRef<MediaStream | null>(null);
  const chunksRef = useRef<Blob[]>([]);
  const timerRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const startTimeRef = useRef(0);
  const maxTimerRef = useRef<ReturnType<typeof setTimeout> | null>(null);
  const onCompleteRef = useRef(onComplete);
  onCompleteRef.current = onComplete;
  const pendingStopRef = useRef(false);

  const isSupported =
    typeof navigator !== "undefined" &&
    typeof navigator.mediaDevices !== "undefined" &&
    typeof MediaRecorder !== "undefined" &&
    !!getSupportedMimeType();

  const cleanup = useCallback(() => {
    if (timerRef.current) {
      clearInterval(timerRef.current);
      timerRef.current = null;
    }
    if (maxTimerRef.current) {
      clearTimeout(maxTimerRef.current);
      maxTimerRef.current = null;
    }
    if (streamRef.current) {
      streamRef.current.getTracks().forEach((track) => track.stop());
      streamRef.current = null;
    }
    mediaRecorderRef.current = null;
    chunksRef.current = [];
  }, []);

  const startRecording = useCallback(() => {
    if (state !== "idle") return;

    const mimeType = getSupportedMimeType();
    if (!mimeType) {
      setError("Audio recording is not supported in this browser");
      return;
    }

    setError(null);
    pendingStopRef.current = false;
    setState("requesting");

    navigator.mediaDevices
      .getUserMedia({ audio: true })
      .then((stream) => {
        streamRef.current = stream;
        chunksRef.current = [];

        const recorder = new MediaRecorder(stream, { mimeType });
        mediaRecorderRef.current = recorder;

        recorder.ondataavailable = (event) => {
          if (event.data.size > 0) {
            chunksRef.current.push(event.data);
          }
        };

        recorder.onstop = async () => {
          setState("processing");
          const chunks = chunksRef.current;
          cleanup();

          if (chunks.length === 0) {
            setState("idle");
            setElapsed(0);
            return;
          }

          try {
            const blob = new Blob(chunks, { type: mimeType });
            const base64 = await blobToBase64(blob);
            onCompleteRef.current({ base64, mimeType });
          } catch (err) {
            uiLogger.error("Failed to process audio recording", { error: err });
            setError("Failed to process recording");
          }

          setState("idle");
          setElapsed(0);
        };

        recorder.onerror = () => {
          uiLogger.error("MediaRecorder error");
          setError("Recording failed");
          cleanup();
          setState("idle");
          setElapsed(0);
        };

        // Start recording
        recorder.start(250); // Collect data every 250ms
        startTimeRef.current = Date.now();
        setState("recording");

        // If user already released the button during permission request, stop immediately
        if (pendingStopRef.current) {
          pendingStopRef.current = false;
          recorder.stop();
          return;
        }

        // Elapsed timer (updates UI every 100ms)
        timerRef.current = setInterval(() => {
          const secs = (Date.now() - startTimeRef.current) / 1000;
          setElapsed(Math.min(secs, MAX_RECORDING_SECONDS));
        }, 100);

        // Auto-stop at max duration
        maxTimerRef.current = setTimeout(() => {
          if (mediaRecorderRef.current?.state === "recording") {
            mediaRecorderRef.current.stop();
          }
        }, MAX_RECORDING_SECONDS * 1000);
      })
      .catch((err) => {
        uiLogger.error("Microphone access denied", { error: err });
        if (err instanceof DOMException && err.name === "NotAllowedError") {
          setError("Microphone access denied. Please allow microphone access.");
        } else {
          setError("Could not access microphone");
        }
        setState("idle");
      });
  }, [state, cleanup]);

  const stopRecording = useCallback(() => {
    if (mediaRecorderRef.current?.state === "recording") {
      mediaRecorderRef.current.stop();
    } else if (state === "requesting") {
      // User released before mic permission was granted — schedule stop
      pendingStopRef.current = true;
    }
  }, [state]);

  const cancelRecording = useCallback(() => {
    // Clear chunks before stopping so onstop produces nothing
    chunksRef.current = [];
    if (mediaRecorderRef.current?.state === "recording") {
      mediaRecorderRef.current.stop();
    } else {
      cleanup();
      setState("idle");
      setElapsed(0);
    }
  }, [cleanup]);

  // Cleanup on unmount
  useEffect(() => {
    return () => {
      if (mediaRecorderRef.current?.state === "recording") {
        chunksRef.current = [];
        mediaRecorderRef.current.stop();
      }
      cleanup();
    };
  }, [cleanup]);

  return {
    state,
    elapsed,
    maxDuration: MAX_RECORDING_SECONDS,
    isSupported,
    startRecording,
    stopRecording,
    cancelRecording,
    error,
  };
}
