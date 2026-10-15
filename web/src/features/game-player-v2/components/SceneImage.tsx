import { useState, useEffect } from "react";
import { useGamePlayerContext } from "../context";
import { translateErrorCode } from "@/common/lib/errorHelpers";
import { ErrorCodes } from "@/common/types/errorCodes";
import { config } from "@/config/env";
import type { ImageStatus } from "../types";
import classes from "./GamePlayer.module.css";

interface SceneImageProps {
  messageId: string;
  imagePrompt?: string;
  imageStatus?: ImageStatus;
  imageHash?: string;
  imageErrorCode?: string;
}

/**
 * Renders the image for a game message.
 * Image status and hash are provided by the parent (via useGameSession SSE + polling fallback).
 * This component is a pure renderer — no independent polling.
 * Parent should use key={messageId} to reset state when the message changes.
 */
export function SceneImage({
  messageId,
  imagePrompt,
  imageStatus,
  imageHash,
  imageErrorCode,
}: SceneImageProps) {
  const { openLightbox, disableImageGeneration } = useGamePlayerContext();
  const [hasLoaded, setHasLoaded] = useState(false);
  const [loadFailed, setLoadFailed] = useState(false);

  // Build image URL:
  // - While generating: bare URL, but only once a partial frame exists
  //   (imageHash is the "partial" marker). The server serves partials with
  //   Cache-Control: no-store, so each fetch is the newest frame.
  // - On complete: content-addressed `?v=<hash>` so the browser caches one
  //   stable URL for the life of the image (identical live and after reload).
  //   For rows persisted before image hashes existed, a constant `?v=legacy`
  //   still differs from the bare generating URL (so the final image loads) and
  //   stays stable across reloads (so it is cached once).
  const baseImageUrl = `${config.API_BASE_URL}/messages/${messageId}/image`;
  let imageUrl: string | null = null;
  if (imageStatus === "complete") {
    imageUrl = `${baseImageUrl}?v=${imageHash || "legacy"}`;
  } else if (imageStatus === "generating" && imageHash) {
    imageUrl = baseImageUrl;
  }

  // Notify context of image error. A per-scene "couldn't generate this image"
  // (retry budget exhausted) is NOT a session-wide blocker, so it only renders
  // the inline notice below - it must not disable image generation for the whole
  // session or pop the blocking error modal.
  useEffect(() => {
    if (
      imageStatus === "error" &&
      imageErrorCode &&
      imageErrorCode !== ErrorCodes.IMAGE_GENERATION_UNAVAILABLE
    ) {
      disableImageGeneration(imageErrorCode);
    }
  }, [imageStatus, imageErrorCode, disableImageGeneration]);

  // Image 404'd (e.g. generation failed but hasImage is still true in DB) — hide entirely
  if (loadFailed) return null;

  const showPlaceholder = imageStatus !== "error" && (!imageUrl || !hasLoaded);
  const isPartialImage = imageStatus === "generating" && !!imageUrl;

  const errorInfo =
    imageStatus === "error" && imageErrorCode
      ? translateErrorCode(imageErrorCode)
      : null;
  // imageErrorCode may be a raw error message from SSE (not a known i18n code).
  // Fall back to showing it directly if translateErrorCode didn't produce a result.
  const errorMessage = errorInfo?.message || imageErrorCode || "Image generation failed";

  const handleImageLoad = () => {
    setHasLoaded(true);
  };

  const handleClick = () => {
    if (hasLoaded && imageUrl) {
      openLightbox(imageUrl, imagePrompt);
    }
  };

  if (imageStatus === "error") {
    return (
      <div className={classes.sceneImageWrapper}>
        <div className={classes.imageError}>
          <span className={classes.imageErrorIcon}>⚠️</span>
          <span className={classes.imageErrorText}>
            {errorMessage}
          </span>
        </div>
      </div>
    );
  }

  return (
    <div
      className={classes.sceneImageWrapper}
      onClick={handleClick}
      role={hasLoaded ? "button" : undefined}
      tabIndex={hasLoaded ? 0 : undefined}
      onKeyDown={(e) => {
        if (hasLoaded && (e.key === "Enter" || e.key === " ")) {
          e.preventDefault();
          handleClick();
        }
      }}
    >
      {showPlaceholder && <div className={classes.imagePlaceholder} />}
      {imageUrl && (
        <img
          src={imageUrl}
          alt={
            imagePrompt ||
            (isPartialImage ? "Generating scene..." : "Scene illustration")
          }
          className={`${classes.sceneImage} ${isPartialImage ? classes.partialImage : ""}`}
          onLoad={handleImageLoad}
          onError={() => setLoadFailed(true)}
        />
      )}
    </div>
  );
}
