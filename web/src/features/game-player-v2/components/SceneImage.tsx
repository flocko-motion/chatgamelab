import { useState, useEffect } from 'react';
import { useGamePlayerContext } from '../context';
import { useImagePolling } from '../hooks/useImagePolling';
import { translateErrorCode } from '@/common/lib/errorHelpers';
import classes from './GamePlayer.module.css';

interface SceneImageProps {
  messageId: string;
  imagePrompt?: string;
  isGenerating?: boolean;
}

export function SceneImage({ messageId, imagePrompt }: SceneImageProps) {
  const { openLightbox, disableImageGeneration } = useGamePlayerContext();
  const [loadedSrc, setLoadedSrc] = useState<string | null>(null);
  const [errorHandled, setErrorHandled] = useState(false);
  const [pollingComplete, setPollingComplete] = useState(false);

  // Reset polling state when messageId changes
  /* eslint-disable react-hooks/set-state-in-effect -- Intentional: reset state when messageId changes */
  useEffect(() => {
    setPollingComplete(false);
    setLoadedSrc(null);
    setErrorHandled(false);
  }, [messageId]);

  // Poll for image updates until complete (not just until first partial image loads)
  const { imageUrl, isComplete, hasError, errorCode } = useImagePolling({
    messageId,
    enabled: !pollingComplete, // Keep polling until we mark it complete
  });

  // Stop polling when image is complete or has error
  if ((isComplete || hasError) && !pollingComplete) {
    setPollingComplete(true);
  }

  // Notify context of image error (to disable future image generation and show modal)
  if (hasError && errorCode && !errorHandled) {
    setErrorHandled(true);
    disableImageGeneration(errorCode);
  }

  const imgLoaded = !!imageUrl && loadedSrc === imageUrl;
  const showPlaceholder = !hasError && (!imageUrl || !imgLoaded);
  const isPartialImage = !isComplete && !!imageUrl;

  // Get translated error message
  const errorInfo = hasError && errorCode ? translateErrorCode(errorCode) : null;

  const handleImageLoad = () => {
    if (imageUrl) {
      setLoadedSrc(imageUrl);
    }
  };

  const handleClick = () => {
    if (imgLoaded && imageUrl) {
      openLightbox(imageUrl, imagePrompt);
    }
  };

  // Show error state
  if (hasError) {
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
      role={imgLoaded ? 'button' : undefined}
      tabIndex={imgLoaded ? 0 : undefined}
      onKeyDown={(e) => {
        if (imgLoaded && (e.key === 'Enter' || e.key === ' ')) {
          e.preventDefault();
          handleClick();
        }
      }}
    >
      {showPlaceholder && <div className={classes.imagePlaceholder} />}
      {imageUrl && (
        <img
          src={imageUrl}
          alt={imagePrompt || (isPartialImage ? 'Generating scene...' : 'Scene illustration')}
          className={`${classes.sceneImage} ${isPartialImage ? classes.partialImage : ''}`}
          onLoad={handleImageLoad}
        />
      )}
    </div>
  );
}
