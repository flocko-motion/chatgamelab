import { useTranslation } from 'react-i18next';
import classes from './GamePlayer.module.css';

export function TypingIndicator() {
  const { t } = useTranslation('common');
  
  return (
    <div className={classes.typingIndicator}>
      <div className={classes.typingDots}>
        <span className={classes.typingDot} />
        <span className={classes.typingDot} />
        <span className={classes.typingDot} />
      </div>
      <span className={classes.typingText}>{t('gamePlayer.typing')}</span>
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
