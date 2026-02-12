import { useState, useRef, useCallback, useEffect } from "react";
import { ActionIcon, Tooltip, Loader } from "@mantine/core";
import { IconVolume, IconPlayerStop } from "@tabler/icons-react";
import { config } from "@/config/env";
import { apiLogger } from "@/config/logger";

type AudioState = "idle" | "loading" | "playing";

interface AudioPlayButtonProps {
  messageId: string;
  /** Current audio status from the streaming session */
  audioStatus?: "loading" | "ready";
  /** Blob URL from streamed audio data (set by SSE consumer) */
  audioBlobUrl?: string;
}

/**
 * AudioPlayButton - Plays audio narration for a game message.
 *
 * Uses the streamed audioBlobUrl when available (live sessions).
 * Falls back to fetching from GET /messages/{id}/audio (resumed sessions).
 */
export function AudioPlayButton({
  messageId,
  audioStatus,
  audioBlobUrl,
}: AudioPlayButtonProps) {
  const [state, setState] = useState<AudioState>("idle");
  const audioRef = useRef<HTMLAudioElement | null>(null);
  const fetchedUrlRef = useRef<string | null>(null);

  const stop = useCallback(() => {
    if (audioRef.current) {
      audioRef.current.pause();
      audioRef.current.currentTime = 0;
    }
    setState("idle");
  }, []);

  const play = useCallback(async () => {
    if (state === "playing") {
      stop();
      return;
    }

    if (audioStatus === "loading") return;

    setState("loading");

    try {
      // Prefer streamed blob URL, then cached fetch, then fetch from API
      let url = audioBlobUrl || fetchedUrlRef.current;
      if (!url) {
        const response = await fetch(
          `${config.API_BASE_URL}/messages/${messageId}/audio`,
        );
        if (!response.ok) {
          throw new Error(`Audio fetch failed: ${response.status}`);
        }
        const blob = await response.blob();
        url = URL.createObjectURL(blob);
        fetchedUrlRef.current = url;
      }

      if (!audioRef.current) {
        audioRef.current = new Audio();
        audioRef.current.addEventListener("ended", () => setState("idle"));
        audioRef.current.addEventListener("error", () => {
          apiLogger.error("Audio playback error", { messageId });
          setState("idle");
        });
      }

      audioRef.current.src = url;
      await audioRef.current.play();
      setState("playing");
    } catch (error) {
      apiLogger.error("Failed to play audio", { messageId, error });
      setState("idle");
    }
  }, [messageId, state, audioStatus, audioBlobUrl, stop]);

  // Auto-play when streamed audio becomes available
  const prevBlobUrlRef = useRef<string | undefined>(undefined);
  useEffect(() => {
    if (audioBlobUrl && !prevBlobUrlRef.current && state === "idle") {
      play();
    }
    prevBlobUrlRef.current = audioBlobUrl;
  }, [audioBlobUrl]); // eslint-disable-line react-hooks/exhaustive-deps

  // Still generating - show spinner
  if (audioStatus === "loading") {
    return (
      <Tooltip label="Audio generating..." position="left">
        <ActionIcon
          variant="subtle"
          color="gray"
          size="sm"
          radius="xl"
          disabled
          aria-label="Audio generating"
        >
          <Loader size={14} />
        </ActionIcon>
      </Tooltip>
    );
  }

  return (
    <Tooltip
      label={state === "playing" ? "Stop" : "Play narration"}
      position="left"
    >
      <ActionIcon
        variant="subtle"
        color={state === "playing" ? "violet" : "gray"}
        size="sm"
        radius="xl"
        onClick={play}
        aria-label={state === "playing" ? "Stop audio" : "Play audio narration"}
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
