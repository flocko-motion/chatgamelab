import type { SceneMessage } from '../types';
import { useGamePlayerContext } from '../context';
import { useGameTheme } from '../theme';
import { SceneImage } from './SceneImage';
import { StreamingIndicator } from './TypingIndicator';
import classes from './GamePlayer.module.css';

interface SceneCardProps {
  message: SceneMessage;
  showImages: boolean;
}

const FONT_SIZE_MAP = {
  xs: 'var(--mantine-font-size-xs)',
  sm: 'var(--mantine-font-size-sm)',
  md: 'var(--mantine-font-size-md)',
  lg: 'var(--mantine-font-size-lg)',
  xl: 'var(--mantine-font-size-xl)',
  '2xl': '1.375rem',
  '3xl': '1.625rem',
} as const;

export function SceneCard({ message, showImages }: SceneCardProps) {
  const { fontSize } = useGamePlayerContext();
  const { theme } = useGameTheme();
  const { text, imageUrl, imagePrompt, isStreaming, isImageLoading } = message;

  const hasImage = showImages && (imageUrl || isImageLoading || isStreaming);
  const isMonochrome = theme.gameMessage?.monochrome ?? false;

  return (
    <div className={`${classes.sceneCard} ${isMonochrome ? classes.monochrome : ''}`}>
      <div className={`${classes.gameScene} ${!hasImage ? classes.noImage : ''}`}>
        {hasImage && (
          <SceneImage
            imageUrl={imageUrl}
            imagePrompt={imagePrompt}
            isLoading={isImageLoading || (isStreaming && !imageUrl)}
          />
        )}
        <div className={classes.sceneContent}>
          <div 
            className={classes.narrativeText}
            style={{ fontSize: FONT_SIZE_MAP[fontSize] }}
          >
            {text}
            {isStreaming && text.length > 0 && <StreamingIndicator />}
          </div>
        </div>
      </div>
    </div>
  );
}
