import type { SceneMessage } from '../types';
import { StreamingIndicator } from './TypingIndicator';
import classes from './GamePlayer.module.css';

interface SystemMessageProps {
  message: SceneMessage;
}

export function SystemMessage({ message }: SystemMessageProps) {
  const { text, isStreaming } = message;

  return (
    <div className={classes.systemMessage}>
      {text}
      {isStreaming && text.length > 0 && <StreamingIndicator />}
    </div>
  );
}
