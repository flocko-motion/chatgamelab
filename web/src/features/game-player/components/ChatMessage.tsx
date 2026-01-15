import { useMemo, useState } from 'react';
import { Text } from '@mantine/core';
import type { ChatMessage as ChatMessageType } from '../types';
import classes from './GamePlayer.module.css';

interface ChatMessageProps {
  message: ChatMessageType;
  showImages?: boolean;
}

export function ChatMessage({ message, showImages = true }: ChatMessageProps) {
  const { type, text, imageUrl, isStreaming, imagePrompt, isImageLoading } = message;

  const [loadedSrc, setLoadedSrc] = useState<string | null>(null);
  const [retryState, setRetryState] = useState<{ baseUrl?: string; attempt: number; nonce: number }>(() => ({
    baseUrl: imageUrl,
    attempt: 0,
    nonce: 0,
  }));

  // Show image section for: existing image, loading image, or streaming game messages
  const hasImageContent = imageUrl || isImageLoading || (type === 'game' && isStreaming);
  const showImage = showImages && hasImageContent;

  const effectiveAttempt = retryState.baseUrl === imageUrl ? retryState.attempt : 0;
  const effectiveNonce = retryState.baseUrl === imageUrl ? retryState.nonce : 0;

  const imgSrc = useMemo(() => {
    if (!imageUrl) return undefined;
    if (effectiveAttempt === 0) return imageUrl;

    const join = imageUrl.includes('?') ? '&' : '?';
    return `${imageUrl}${join}t=${effectiveNonce}&r=${effectiveAttempt}`;
  }, [effectiveAttempt, effectiveNonce, imageUrl]);

  const imgLoaded = !!imgSrc && loadedSrc === imgSrc;
  const showPlaceholder = showImage && (!imgSrc || !imgLoaded);

  const handleImageError = () => {
    if (!imageUrl) return;
    // Poll every 2 seconds for up to 30 attempts (1 minute total)
    if (effectiveAttempt >= 30) return;

    const nextAttempt = effectiveAttempt + 1;
    window.setTimeout(() => {
      setRetryState({ baseUrl: imageUrl, attempt: nextAttempt, nonce: Date.now() });
    }, 2000);
  };

  const streamingIndicator = isStreaming && text.length > 0 && (
    <span className={classes.streamingIndicator}>
      <span className={classes.streamingDot} />
      <span className={classes.streamingDot} />
      <span className={classes.streamingDot} />
    </span>
  );

  // Player message - simple bubble on the right
  if (type === 'player') {
    return (
      <div className={classes.messageWrapper} data-type="player">
        <div className={classes.playerBubble}>
          <Text size="sm" style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
            {text}
          </Text>
        </div>
      </div>
    );
  }

  // System message - dashed border bubble on the left
  if (type === 'system') {
    return (
      <div className={classes.messageWrapper} data-type="system">
        <div className={classes.systemBubble}>
          <Text size="sm" style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
            {text}
            {streamingIndicator}
          </Text>
        </div>
      </div>
    );
  }

  // Game message - bubble with optional image
  return (
    <div className={classes.messageWrapper} data-type="game">
      <div className={classes.gameBubble}>
        {showImage && (
          <div className={classes.imageSection}>
            {showPlaceholder && <div className={classes.imagePlaceholder} />}
            {imgSrc && (
              <img
                src={imgSrc}
                alt={imagePrompt || 'Game scene'}
                style={{ display: imgLoaded ? 'block' : 'none' }}
                onLoad={() => setLoadedSrc(imgSrc)}
                onError={handleImageError}
              />
            )}
          </div>
        )}
        <Text size="sm" style={{ whiteSpace: 'pre-wrap', wordBreak: 'break-word' }}>
          {text}
          {streamingIndicator}
        </Text>
      </div>
    </div>
  );
}
