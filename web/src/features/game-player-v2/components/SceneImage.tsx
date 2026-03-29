import { useState, useEffect } from "react";
import { useGamePlayerContext } from "../context";
import { translateErrorCode } from "@/common/lib/errorHelpers";
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
  // - During generation: stable URL (no hash) so the <img> element stays mounted
  //   and the browser just refreshes it without restarting CSS animations.
  // - On complete: use hash for cache-busting to ensure final image is shown.
  const baseImageUrl = `${config.API_BASE_URL}/messages/${messageId}/image`;
  const imageUrl = imageHash
    ? imageStatus === "generating"
      ? baseImageUrl
      : `${baseImageUrl}?v=${imageHash}`
    : null;

  // Notify context of image error
  useEffect(() => {
    if (imageStatus === "error" && imageErrorCode) {
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
