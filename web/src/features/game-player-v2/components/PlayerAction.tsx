import { useGameTheme } from '../theme';
import { translateErrorCode } from '@/common/lib/errorHelpers';
import classes from './GamePlayer.module.css';

interface PlayerActionProps {
  text: string;
  error?: string;
  errorCode?: string;
  onRetry?: () => void;
}

// Map indicator type to display character
const INDICATOR_CHARS: Record<string, string> = {
  dot: '•',
  chevron: '>',
  pipe: '|',
  cursor: '▌',
  underscore: '_',
  none: '',
};

export function PlayerAction({ text, error, errorCode, onRetry }: PlayerActionProps) {
  const { theme } = useGameTheme();
  const indicator = theme.player.indicator ?? 'chevron';
  const indicatorBlink = theme.player.indicatorBlink ?? false;

  const bubbleClasses = [
    classes.playerActionBubble,
    indicator === 'none' && classes.noIndicator,
    error && classes.playerActionError,
  ].filter(Boolean).join(' ');

  const indicatorClasses = [
    classes.playerIndicator,
    indicatorBlink && classes.indicatorBlink,
  ].filter(Boolean).join(' ');

  const indicatorChar = INDICATOR_CHARS[indicator] || '>';

  const errorInfo = error && errorCode ? translateErrorCode(errorCode) : null;

  return (
    <div className={classes.playerAction}>
      <div className={bubbleClasses}>
        {indicator !== 'none' && (
          <span className={indicatorClasses}>{indicatorChar}</span>
        )}
        <span className={classes.playerActionText}>{text}</span>
      </div>
      {error && (
        <div className={classes.playerActionErrorInfo}>
          <span className={classes.playerActionErrorText}>
            ⚠️ {errorInfo?.message || error}
          </span>
          {onRetry && (
            <button
              className={classes.playerActionRetryButton}
              onClick={onRetry}
              type="button"
            >
              ↻ Retry
            </button>
          )}
        </div>
      )}
    </div>
  );
}
