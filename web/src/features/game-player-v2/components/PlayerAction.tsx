import classes from './GamePlayer.module.css';

interface PlayerActionProps {
  text: string;
}

export function PlayerAction({ text }: PlayerActionProps) {
  return (
    <div className={classes.playerAction}>
      <div className={classes.playerActionBubble}>
        <span className={classes.playerActionPrefix}>&gt;</span>
        <span className={classes.playerActionText}>{text}</span>
      </div>
    </div>
  );
}
