import { useGameTheme } from '../theme';
import classes from './GamePlayer.module.css';

interface PlayerActionProps {
  text: string;
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

export function PlayerAction({ text }: PlayerActionProps) {
  const { theme } = useGameTheme();
  const indicator = theme.player.indicator ?? 'chevron';
  const indicatorBlink = theme.player.indicatorBlink ?? false;

  const bubbleClasses = [
    classes.playerActionBubble,
    indicator === 'none' && classes.noIndicator,
  ].filter(Boolean).join(' ');

  const indicatorClasses = [
    classes.playerIndicator,
    indicatorBlink && classes.indicatorBlink,
  ].filter(Boolean).join(' ');

  const indicatorChar = INDICATOR_CHARS[indicator] || '>';

  return (
    <div className={classes.playerAction}>
      <div className={bubbleClasses}>
        {indicator !== 'none' && (
          <span className={indicatorClasses}>{indicatorChar}</span>
        )}
        <span className={classes.playerActionText}>{text}</span>
      </div>
    </div>
  );
}
