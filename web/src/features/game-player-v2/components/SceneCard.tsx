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
  const cornerStyle = theme.corners?.style ?? 'brackets';
  const showDropCap = theme.gameMessage?.dropCap ?? false;

  // Map corner style to CSS class
  const cornerClassMap: Record<string, string> = {
    brackets: classes.cornerBrackets,
    flourish: classes.cornerFlourish,
    arrows: classes.cornerArrows,
    dots: classes.cornerDots,
    none: classes.cornerNone,
  };
  const cornerClass = cornerClassMap[cornerStyle] || classes.cornerBrackets;

  const narrativeClasses = [
    classes.narrativeText,
    showDropCap && classes.narrativeTextDropCap,
  ].filter(Boolean).join(' ');

  return (
    <div className={classes.sceneCard}>
      <div className={`${classes.gameScene} ${cornerClass} ${!hasImage ? classes.noImage : ''}`}>
        {hasImage && (
          <SceneImage
            imageUrl={imageUrl}
            imagePrompt={imagePrompt}
            isLoading={isImageLoading || (isStreaming && !imageUrl)}
          />
        )}
        <div className={classes.sceneContent}>
          <div 
            className={narrativeClasses}
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
