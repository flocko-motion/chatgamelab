import { useGameTheme } from '../theme';
import classes from './GamePlayer.module.css';

interface PlayerActionProps {
  text: string;
}

export function PlayerAction({ text }: PlayerActionProps) {
  const { theme } = useGameTheme();
  const showChevron = theme.player.showChevron;
  const isMonochrome = theme.player.monochrome;

  const bubbleClasses = [
    classes.playerActionBubble,
    !showChevron && classes.noChevron,
    isMonochrome && classes.playerMonochrome,
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
