import classes from './GamePlayer.module.css';

export function SceneDivider() {
  return (
    <div className={classes.sceneDivider}>
      <div className={classes.dividerLine} />
      <div className={classes.dividerDot} />
      <div className={classes.dividerLine} />
    </div>
  );
}
