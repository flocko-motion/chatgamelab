import { useState, useRef, useCallback, useEffect } from "react";
import { ActionIcon, Tooltip, Loader } from "@mantine/core";
import { IconVolume, IconPlayerStop } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { apiLogger } from "@/config/logger";
import { useGamePlayerContext } from "../context";
import { stopAllAudio, registerAudioSource } from "../lib/audioManager";

type AudioState = "idle" | "loading" | "playing";

interface AudioPlayButtonProps {
  messageId: string;
  /** Current audio status from the streaming session */
  audioStatus?: "loading" | "ready";
  /** Blob URL from streamed audio data (set by SSE consumer) */
  audioBlobUrl?: string;
  /** Global narration mute state from player settings */
  isAudioMuted: boolean;
  /**
   * Play once as soon as possible after mounting. Set for the latest scene only, so that
   * switching narration on reads out the current scene and nothing else.
   */
  autoPlay?: boolean;
}

/**
 * AudioPlayButton - Plays audio narration for a game message.
 *
 * Uses the streamed audioBlobUrl when available (live sessions with narration on).
 * Otherwise loads it via loadMessageAudio, which generates the audio on the backend
 * if the scene was played while narration was off.
 */
export function AudioPlayButton({
  messageId,
  audioStatus,
  audioBlobUrl,
  isAudioMuted,
  autoPlay,
}: AudioPlayButtonProps) {
  const { t } = useTranslation("common");
  const { loadMessageAudio } = useGamePlayerContext();
  const [state, setState] = useState<AudioState>("idle");
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const fetchedUrlRef = useRef<string | null>(null);
  const unregisterRef = useRef<(() => void) | null>(null);
  // Bumped on every start and stop; a start still awaiting audio gives up when it is outdated.
  const attemptRef = useRef(0);
  const hasAutoPlayedRef = useRef(false);

  const stop = useCallback(() => {
    attemptRef.current++;
    if (audioRef.current) {
      audioRef.current.pause();
      audioRef.current.currentTime = 0;
    }
    unregisterRef.current?.();
    unregisterRef.current = null;
    setState("idle");
  }, []);

  const start = useCallback(async () => {
    if (audioStatus === "loading" || isAudioMuted) return;
    hasAutoPlayedRef.current = true;

    // Claim the audio output before awaiting anything: whatever starts next (another scene,
    // voice recording) stops this one, even while it is still loading.
    stopAllAudio();
    const attempt = ++attemptRef.current;
    unregisterRef.current = registerAudioSource(stop);
    setState("loading");

    try {
      // Prefer streamed blob URL, then cached audio, then load (and generate) it
      let url = audioBlobUrl || fetchedUrlRef.current;
      if (!url) {
        const blob = await loadMessageAudio(messageId);
        url = URL.createObjectURL(blob);
        fetchedUrlRef.current = url;
      }
      if (attempt !== attemptRef.current) return;

      if (!audioRef.current) {
        audioRef.current = new Audio();
        audioRef.current.addEventListener("ended", () => {
          unregisterRef.current?.();
          unregisterRef.current = null;
          setState("idle");
        });
        audioRef.current.addEventListener("error", () => {
          apiLogger.error("Audio playback error", { messageId });
          unregisterRef.current?.();
          unregisterRef.current = null;
          setState("idle");
        });
      }

      audioRef.current.src = url;
      await audioRef.current.play();
      if (attempt !== attemptRef.current) {
        audioRef.current.pause();
        return;
      }
      setState("playing");
    } catch (error) {
      // Stopped while starting: play() rejects when paused before it began
      if (attempt !== attemptRef.current) return;
      apiLogger.error("Failed to play audio", { messageId, error });
      unregisterRef.current?.();
      unregisterRef.current = null;
      setState("idle");
    }
  }, [
    messageId,
    audioStatus,
    audioBlobUrl,
    isAudioMuted,
    loadMessageAudio,
    stop,
  ]);

  const toggle = useCallback(() => {
    if (state === "idle") {
      start();
    } else {
      stop();
    }
  }, [state, start, stop]);

  // Stop and clean up on unmount (muting unmounts all buttons)
  useEffect(() => {
    return () => {
      stop();
      const fetchedUrl = fetchedUrlRef.current;
      if (fetchedUrl) {
        URL.revokeObjectURL(fetchedUrl);
        fetchedUrlRef.current = null;
      }
    };
  }, [stop]);

  // Auto-play streamed audio that arrives while this button is shown. Audio that already
  // existed when the button appeared (e.g. narration switched on later) is not played.
  const prevBlobUrlRef = useRef(audioBlobUrl);
  useEffect(() => {
    if (
      !isAudioMuted &&
      audioBlobUrl &&
      !prevBlobUrlRef.current &&
      state === "idle"
    ) {
      hasAutoPlayedRef.current = true;
      // Deferred so the state update does not happen within the effect itself
      queueMicrotask(() => void start());
    }
    prevBlobUrlRef.current = audioBlobUrl;
  }, [audioBlobUrl, isAudioMuted]); // eslint-disable-line react-hooks/exhaustive-deps

  // Latest scene: play once when narration is switched on, or once its text is complete
  useEffect(() => {
    if (
      autoPlay &&
      !hasAutoPlayedRef.current &&
      audioStatus !== "loading" &&
      state === "idle"
    ) {
      hasAutoPlayedRef.current = true;
      queueMicrotask(() => void start());
    }
  }, [autoPlay, audioStatus]); // eslint-disable-line react-hooks/exhaustive-deps

  // Still generating - show spinner
  if (audioStatus === "loading") {
    return (
      <Tooltip label={t("gamePlayer.narration.generating")} position="left">
        <ActionIcon
          variant="subtle"
          color="gray"
          size="sm"
          radius="xl"
          disabled
          aria-label={t("gamePlayer.narration.generating")}
        >
          <Loader size={14} />
        </ActionIcon>
      </Tooltip>
    );
  }

  const label =
    state === "idle"
      ? t("gamePlayer.narration.play")
      : t("gamePlayer.narration.stop");

  return (
    <Tooltip label={label} position="left">
      <ActionIcon
        variant="subtle"
        color={state === "playing" ? "violet" : "gray"}
        size="sm"
        radius="xl"
        onClick={toggle}
        aria-label={label}
        loading={state === "loading"}
      >
        {state === "playing" ? (
          <IconPlayerStop size={16} stroke={1.5} />
        ) : (
          <IconVolume size={16} stroke={1.5} />
        )}
      </ActionIcon>
    </Tooltip>
  );
}
