import type { ObjStatusField } from '@/api/generated';
import type { SceneMessage } from '../types';
import { useGamePlayerContext } from '../context';
import { useGameTheme } from '../theme';
import { SceneImage } from './SceneImage';
import { StreamingIndicator } from './TypingIndicator';
import { DebugPanel } from './DebugPanel';
import { StatusChangeIndicator } from './StatusChangeIndicator';
import classes from './GamePlayer.module.css';

interface SceneCardProps {
  message: SceneMessage;
  showImages: boolean;
  previousStatusFields?: ObjStatusField[];
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

export function SceneCard({ message, showImages, previousStatusFields }: SceneCardProps) {
  const { fontSize, debugMode } = useGamePlayerContext();
  const { theme } = useGameTheme();
  const { text, imageUrl, imagePrompt, isStreaming, isImageLoading } = message;

  const hasImage = showImages && (imageUrl || isImageLoading || isStreaming);
  const cornerStyle = theme.corners?.style ?? 'brackets';
  const showDropCap = theme.gameMessage?.dropCap ?? false;
  
  // Corner positions (default: top-left and bottom-right)
  const positions = theme.corners?.positions ?? {
    topLeft: true,
    topRight: false,
    bottomLeft: false,
    bottomRight: true,
  };
  const cornerBlink = theme.corners?.blink ?? false;

  // Map corner style to CSS class prefix
  const cornerStyleClass = cornerStyle !== 'none' ? classes[`corner${cornerStyle.charAt(0).toUpperCase() + cornerStyle.slice(1)}`] : '';

  const narrativeClasses = [
    classes.narrativeText,
    showDropCap && classes.narrativeTextDropCap,
  ].filter(Boolean).join(' ');

  const sceneClasses = [
    classes.gameScene,
    !hasImage && classes.noImage,
  ].filter(Boolean).join(' ');

  // Render corner decoration element
  const renderCorner = (position: 'topLeft' | 'topRight' | 'bottomLeft' | 'bottomRight') => {
    if (cornerStyle === 'none' || !positions[position]) return null;
    const positionClass = classes[`corner${position.charAt(0).toUpperCase() + position.slice(1)}`];
    const blinkClass = cornerBlink ? classes.cornerBlink : '';
    return (
      <span 
        className={`${classes.cornerDecor} ${cornerStyleClass} ${positionClass} ${blinkClass}`.trim()}
        aria-hidden="true"
      />
    );
  };

  return (
    <div className={classes.sceneCard}>
      <div className={sceneClasses}>
        {renderCorner('topLeft')}
        {renderCorner('topRight')}
        {renderCorner('bottomLeft')}
        {renderCorner('bottomRight')}
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
      {previousStatusFields && message.statusFields && (
        <StatusChangeIndicator 
          currentFields={message.statusFields} 
          previousFields={previousStatusFields} 
        />
      )}
      {debugMode && <DebugPanel message={message} />}
    </div>
  );
}
