import { useGameTheme } from '../theme';
import classes from './GamePlayer.module.css';

export function TypingIndicator() {
  const { theme } = useGameTheme();
  
  return (
    <div className={classes.typingIndicator}>
      <div className={classes.typingDots}>
        <span className={classes.typingDot} />
        <span className={classes.typingDot} />
        <span className={classes.typingDot} />
      </div>
      <span className={classes.typingText}>{theme.thinking.text}</span>
    </div>
  );
}

export function StreamingIndicator() {
  return (
    <span className={classes.streamingIndicator}>
      <span className={classes.streamingDot} />
      <span className={classes.streamingDot} />
      <span className={classes.streamingDot} />
    </span>
  );
}
