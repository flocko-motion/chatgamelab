import { useState, useMemo } from 'react';
import { useGamePlayerContext } from '../context';
import classes from './GamePlayer.module.css';

interface SceneImageProps {
  imageUrl?: string;
  imagePrompt?: string;
  isLoading?: boolean;
}

const MAX_RETRIES = 30;
const RETRY_INTERVAL_MS = 2000;

export function SceneImage({ imageUrl, imagePrompt, isLoading }: SceneImageProps) {
  const { openLightbox } = useGamePlayerContext();
  
  const [loadedSrc, setLoadedSrc] = useState<string | null>(null);
  const [retryState, setRetryState] = useState<{ baseUrl?: string; attempt: number; nonce: number }>({
    baseUrl: imageUrl,
    attempt: 0,
    nonce: 0,
  });

  const effectiveAttempt = retryState.baseUrl === imageUrl ? retryState.attempt : 0;
  const effectiveNonce = retryState.baseUrl === imageUrl ? retryState.nonce : 0;

  const imgSrc = useMemo(() => {
    if (!imageUrl) return undefined;
    if (effectiveAttempt === 0) return imageUrl;
    const join = imageUrl.includes('?') ? '&' : '?';
    return `${imageUrl}${join}t=${effectiveNonce}&r=${effectiveAttempt}`;
  }, [effectiveAttempt, effectiveNonce, imageUrl]);

  const imgLoaded = !!imgSrc && loadedSrc === imgSrc;
  const showPlaceholder = !imgSrc || !imgLoaded || isLoading;

  const handleImageError = () => {
    if (!imageUrl) return;
    if (effectiveAttempt >= MAX_RETRIES) return;

    const nextAttempt = effectiveAttempt + 1;
    window.setTimeout(() => {
      setRetryState({ baseUrl: imageUrl, attempt: nextAttempt, nonce: Date.now() });
    }, RETRY_INTERVAL_MS);
  };

  const handleImageLoad = () => {
    if (imgSrc) {
      setLoadedSrc(imgSrc);
    }
  };

  const handleClick = () => {
    if (imgLoaded && imgSrc) {
      openLightbox(imgSrc, imagePrompt);
    }
  };

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
      {imgSrc && (
        <img
          src={imgSrc}
          alt={imagePrompt || 'Scene illustration'}
          className={classes.sceneImage}
          style={{ display: imgLoaded ? 'block' : 'none' }}
          onLoad={handleImageLoad}
          onError={handleImageError}
        />
      )}
    </div>
  );
}
