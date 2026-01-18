import { useGameTheme } from '../theme';
import classes from './GamePlayer.module.css';

interface PlayerActionProps {
  text: string;
}

export function PlayerAction({ text }: PlayerActionProps) {
  const { theme } = useGameTheme();
  const showChevron = theme.player.showChevron;

  const bubbleClasses = [
    classes.playerActionBubble,
    !showChevron && classes.noChevron,
  ].filter(Boolean).join(' ');

  return (
    <div className={classes.playerAction}>
      <div className={bubbleClasses}>
        <span className={classes.playerActionPrefix}>&gt;</span>
        <span className={classes.playerActionText}>{text}</span>
      </div>
    </div>
  );
}
