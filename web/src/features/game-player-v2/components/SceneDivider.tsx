import { useGameTheme } from '../theme';
import type { DividerStyle } from '../theme/types';
import classes from './GamePlayer.module.css';

const DIVIDER_SYMBOLS: Record<DividerStyle, string> = {
  dot: '•',
  line: '—',
  dots: '• • •',
  diamond: '◆',
  star: '✦',
  dash: '---',
  none: '',
};

export function SceneDivider() {
  const { theme, cssVars } = useGameTheme();
  const style = theme.divider?.style ?? 'dot';
  
  if (style === 'none') {
    return <div className={classes.sceneDividerEmpty} />;
  }
  
  const symbol = DIVIDER_SYMBOLS[style];
  
  return (
    <div className={classes.sceneDivider} style={cssVars}>
      {style === 'line' ? (
        <div className={classes.dividerFullLine} />
      ) : (
        <>
          <div className={classes.dividerLine} />
          <span className={classes.dividerSymbol}>{symbol}</span>
          <div className={classes.dividerLine} />
        </>
      )}
    </div>
  );
}
