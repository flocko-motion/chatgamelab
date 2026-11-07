/**
 * FadeInText effect for AI messages.
 *
 * Words fade in sequentially with a staggered delay.
 * Designed for mystic, romance, fairy themes.
 */

import { useMemo } from 'react';

const KEYFRAMES = `
@keyframes textFadeIn {
  0% {
    opacity: 0;
    transform: translateY(4px);
  }
  100% {
    opacity: 1;
    transform: translateY(0);
  }
}
`;

let keyframesInjected = false;

function ensureKeyframes() {
  if (keyframesInjected) return;
  if (typeof document === 'undefined') return;
  const style = document.createElement('style');
  style.textContent = KEYFRAMES;
  document.head.appendChild(style);
  keyframesInjected = true;
}

interface FadeInTextProps {
  text: string;
}

export function FadeInText({ text }: FadeInTextProps) {
  ensureKeyframes();

  const words = useMemo(() => text.split(/(\s+)/), [text]);

  let wordIndex = 0;

  return (
    <span>
      {words.map((word, i) => {
        if (/^\s+$/.test(word)) {
          return word;
        }
        const delay = Math.min(wordIndex * 0.06, 3);
        wordIndex++;
        return (
          <span
            key={i}
            style={{
              display: 'inline-block',
              opacity: 0,
              animation: `textFadeIn 0.4s ease-out ${delay}s forwards`,
            }}
          >
            {word}
          </span>
        );
      })}
    </span>
  );
}
