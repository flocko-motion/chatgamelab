import { useGameTheme } from '../theme';
import { CARD_BG_COLORS, FONT_COLORS, THEME_COLORS, CARD_BORDER_THICKNESSES } from '../theme/defaults';
import classes from './GamePlayer.module.css';

/** Cursor characters for streaming indicator */
const STREAMING_CURSORS: Record<string, string> = {
  block: '█',
  pipe: '|',
  underscore: '_',
  dots: '', // handled separately with animation
  none: '',
};

export function TypingIndicator() {
  const { theme, cssVars } = useGameTheme();
  
  // Use game message styling
  const bgColorDef = CARD_BG_COLORS[theme.gameMessage.bgColor] || CARD_BG_COLORS.white;
  const fontColor = FONT_COLORS[theme.gameMessage.fontColor] || FONT_COLORS.dark;
  const borderColor = THEME_COLORS[theme.gameMessage.borderColor] || THEME_COLORS.slate;
  const borderThickness = CARD_BORDER_THICKNESSES[theme.cards.borderThickness] || '1px';
  
  const style: React.CSSProperties = {
    ...cssVars,
    background: bgColorDef.alpha,
    color: fontColor,
    borderColor: borderColor.primary,
    borderWidth: borderThickness,
    borderStyle: 'solid',
  };
  
  return (
    <div className={classes.typingIndicator} style={style}>
      <div className={classes.typingDots}>
        <span className={classes.typingDot} />
        <span className={classes.typingDot} />
        <span className={classes.typingDot} />
      </div>
      <span className={classes.typingText} style={{ color: fontColor }}>{theme.thinking.text}</span>
    </div>
  );
}

export function StreamingIndicator() {
  const { theme } = useGameTheme();
  const cursorStyle = theme.thinking.streamingCursor || 'dots';
  
  // For dots, use animated dots
  if (cursorStyle === 'dots') {
    return (
      <span className={classes.streamingIndicator}>
        <span className={classes.streamingDot} />
        <span className={classes.streamingDot} />
        <span className={classes.streamingDot} />
      </span>
    );
  }
  
  // For none, return nothing
  if (cursorStyle === 'none') {
    return null;
  }
  
  // For character cursors (block, pipe, underscore), show blinking character
  const cursorChar = STREAMING_CURSORS[cursorStyle] || '|';
  return (
    <span className={classes.streamingCursor}>{cursorChar}</span>
  );
}
