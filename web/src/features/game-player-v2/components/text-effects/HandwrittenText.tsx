/**
 * HandwrittenText effect for AI messages.
 *
 * Slight random rotation and offset per character, like hand-lettered text.
 * Designed for storybook, adventure, school themes.
 */

import { useMemo } from 'react';

interface HandwrittenTextProps {
  text: string;
}

function seededRandom(seed: number): number {
  const x = Math.sin(seed) * 10000;
  return x - Math.floor(x);
}

export function HandwrittenText({ text }: HandwrittenTextProps) {
  const chars = useMemo(() => {
    return text.split('').map((char, i) => {
      if (char === ' ' || char === '\n') return { char, style: undefined };
      const r1 = seededRandom(i * 7 + 3);
      const r2 = seededRandom(i * 13 + 7);
      const r3 = seededRandom(i * 19 + 11);
      const rotation = (r1 - 0.5) * 4; // -2 to 2 degrees
      const offsetY = (r2 - 0.5) * 2; // -1 to 1px
      const offsetX = (r3 - 0.5) * 0.5; // -0.25 to 0.25px
      return {
        char,
        style: {
          display: 'inline-block' as const,
          transform: `rotate(${rotation}deg) translate(${offsetX}px, ${offsetY}px)`,
        },
      };
    });
  }, [text]);

  return (
    <span>
      {chars.map((c, i) =>
        c.style ? (
          <span key={i} style={c.style}>{c.char}</span>
        ) : (
          c.char
        ),
      )}
    </span>
  );
}
