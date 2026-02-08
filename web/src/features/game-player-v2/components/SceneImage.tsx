import { useState, useEffect, useRef, useCallback } from 'react';
import { useGamePlayerContext } from '../context';
import { translateErrorCode } from '@/common/lib/errorHelpers';
import { apiLogger } from '@/config/logger';
import { config } from '@/config/env';
import type { ImageStatus, MessageStatus } from '../types';
import classes from './GamePlayer.module.css';

const IMAGE_RETRY_POLL_INTERVAL = 2000;
const IMAGE_RETRY_MAX_POLLS = 15;

interface SceneImageProps {
  messageId: string;
  imagePrompt?: string;
  imageStatus?: ImageStatus;
  imageHash?: string;
  imageErrorCode?: string;
}

/**
 * Renders the image for a game message.
 * Image status and hash are provided by the parent (via useGameSession polling).
 * This component just derives the image URL and handles load/error states.
 * Parent should use key={messageId} to reset state when the message changes.
 */
export function SceneImage({ messageId, imagePrompt, imageStatus, imageHash, imageErrorCode }: SceneImageProps) {
  const { openLightbox, disableImageGeneration } = useGamePlayerContext();
  const [hasLoaded, setHasLoaded] = useState(false);
  const [isRetrying, setIsRetrying] = useState(false);
  const [retryHash, setRetryHash] = useState<string | null>(null);
  const retryPollRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const hasRetriedRef = useRef(false);

  // Build image URL:
  // - During generation: stable URL (no hash) so the <img> element stays mounted
  //   and the browser just refreshes it without restarting CSS animations.
  // - On complete: use hash for cache-busting to ensure final image is shown.
  const baseImageUrl = `${config.API_BASE_URL}/messages/${messageId}/image`;
  const imageUrl = imageHash
    ? (imageStatus === 'generating' ? baseImageUrl : `${baseImageUrl}?v=${imageHash}`)
    : null;

  // Notify context of image error
  useEffect(() => {
    if (imageStatus === 'error' && imageErrorCode) {
      disableImageGeneration(imageErrorCode);
    }
  }, [imageStatus, imageErrorCode, disableImageGeneration]);

  // Use retryHash to override the image URL after successful re-generation
  const effectiveImageUrl = retryHash
    ? `${baseImageUrl}?v=${retryHash}`
    : imageUrl;

  const showPlaceholder = imageStatus !== 'error' && (!effectiveImageUrl || !hasLoaded || isRetrying);
  const isPartialImage = imageStatus === 'generating' && !!imageUrl;

  const errorInfo = imageStatus === 'error' && imageErrorCode ? translateErrorCode(imageErrorCode) : null;

  // Clean up retry polling on unmount
  useEffect(() => {
    return () => {
      if (retryPollRef.current) {
        clearInterval(retryPollRef.current);
      }
    };
  }, []);

  // Poll status endpoint to detect backend image re-generation
  const startRetryPolling = useCallback(() => {
    if (retryPollRef.current) return;
    setIsRetrying(true);
    let pollCount = 0;

    retryPollRef.current = setInterval(async () => {
      pollCount++;
      if (pollCount > IMAGE_RETRY_MAX_POLLS) {
        apiLogger.debug('Image retry polling timed out', { messageId });
        if (retryPollRef.current) clearInterval(retryPollRef.current);
        retryPollRef.current = null;
        setIsRetrying(false);
        return;
      }

      try {
        const resp = await fetch(`${config.API_BASE_URL}/messages/${messageId}/status`);
        if (!resp.ok) return;
        const status: MessageStatus = await resp.json();

        if (status.imageStatus === 'complete' && status.imageHash) {
          apiLogger.debug('Image retry: generation complete', { messageId, hash: status.imageHash });
          if (retryPollRef.current) clearInterval(retryPollRef.current);
          retryPollRef.current = null;
          setRetryHash(status.imageHash);
          setIsRetrying(false);
        } else if (status.imageStatus === 'error') {
          apiLogger.debug('Image retry: generation failed', { messageId, error: status.imageError });
          if (retryPollRef.current) clearInterval(retryPollRef.current);
          retryPollRef.current = null;
          setIsRetrying(false);
        }
      } catch {
        // Ignore fetch errors during polling
      }
    }, IMAGE_RETRY_POLL_INTERVAL);
  }, [messageId]);

  const handleImageLoad = () => {
    setHasLoaded(true);
  };

  const handleImageError = () => {
    // Only retry once per component lifecycle
    if (hasRetriedRef.current || imageStatus === 'generating') return;
    hasRetriedRef.current = true;
    apiLogger.debug('Image load failed, starting retry polling', { messageId });
    startRetryPolling();
  };

  const handleClick = () => {
    if (hasLoaded && effectiveImageUrl) {
      openLightbox(effectiveImageUrl, imagePrompt);
    }
  };

  if (imageStatus === 'error') {
    return (
      <div className={classes.sceneImageWrapper}>
        <div className={classes.imageError}>
          <span className={classes.imageErrorIcon}>⚠️</span>
          <span className={classes.imageErrorText}>
            {errorInfo?.message || 'Image generation failed'}
          </span>
        </div>
      </div>
    );
  }

  return (
    <div 
      className={classes.sceneImageWrapper}
      onClick={handleClick}
      role={hasLoaded ? 'button' : undefined}
      tabIndex={hasLoaded ? 0 : undefined}
      onKeyDown={(e) => {
        if (hasLoaded && (e.key === 'Enter' || e.key === ' ')) {
          e.preventDefault();
          handleClick();
        }
      }}
    >
      {showPlaceholder && <div className={classes.imagePlaceholder} />}
      {effectiveImageUrl && (
        <img
          src={effectiveImageUrl}
          alt={imagePrompt || (isPartialImage ? 'Generating scene...' : 'Scene illustration')}
          className={`${classes.sceneImage} ${isPartialImage ? classes.partialImage : ''}`}
          onLoad={handleImageLoad}
          onError={handleImageError}
        />
      )}
    </div>
  );
}
