/**
 * GlitchText effect for AI messages.
 * Inspired by ReactBits (https://reactbits.dev/text-animations/glitch-text)
 * 
 * JS-based: randomly corrupts a few characters at a time on an interval,
 * then restores them, creating a continuous subtle glitch effect.
 * Designed for the glitch/cyberpunk theme.
 */

import { useEffect, useState, useRef, useCallback } from 'react';

const GLITCH_CHARS = '!@#$%^&*()_+-=[]{}|;:<>?/\\~`░▒▓█▄▀■□▪▫';

interface GlitchTextProps {
  text: string;
  /** ms between glitch ticks */
  interval?: number;
  /** fraction of characters to corrupt each tick (0-1) */
  intensity?: number;
}

export function GlitchText({ text, interval = 200, intensity = 0.02 }: GlitchTextProps) {
  const [displayText, setDisplayText] = useState(text);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);
  const textRef = useRef(text);

  // Keep ref in sync
  useEffect(() => {
    textRef.current = text;
  }, [text]);

  const glitch = useCallback(() => {
    const current = textRef.current;
    const chars = GLITCH_CHARS.split('');
    const count = Math.max(1, Math.floor(current.length * intensity));

    const result = current.split('');
    for (let i = 0; i < count; i++) {
      const idx = Math.floor(Math.random() * current.length);
      if (result[idx] !== ' ' && result[idx] !== '\n') {
        result[idx] = chars[Math.floor(Math.random() * chars.length)];
      }
    }
    setDisplayText(result.join(''));
  }, [intensity]);

  useEffect(() => {
    // Update base text immediately when it changes (streaming)
    setDisplayText(text);
  }, [text]);

  useEffect(() => {
    // Alternate between glitched and clean on each tick
    let isGlitched = false;

    intervalRef.current = setInterval(() => {
      if (isGlitched) {
        setDisplayText(textRef.current);
      } else {
        glitch();
      }
      isGlitched = !isGlitched;
    }, interval);

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    };
  }, [interval, glitch]);

  return <span>{displayText}</span>;
}
