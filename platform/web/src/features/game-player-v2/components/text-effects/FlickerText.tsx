/**
 * FlickerText effect for AI messages.
 *
 * Random letters briefly dim/disappear like a dying lightbulb.
 * Designed for horror, noir, mystery themes.
 */

import { useEffect, useState, useRef, useCallback } from "react";

interface FlickerTextProps {
  text: string;
  /** ms between flicker ticks */
  interval?: number;
  /** fraction of characters to dim each tick (0-1) */
  intensity?: number;
}

interface CharState {
  char: string;
  opacity: number;
}

export function FlickerText({
  text,
  interval = 400,
  intensity = 0.03,
}: FlickerTextProps) {
  const [chars, setChars] = useState<CharState[]>(() =>
    text.split("").map((char) => ({ char, opacity: 1 })),
  );
  const textRef = useRef(text);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  // Sync text ref and reset chars when text changes
  useEffect(() => {
    textRef.current = text;
    setChars(text.split("").map((char) => ({ char, opacity: 1 }))); // eslint-disable-line react-hooks/set-state-in-effect -- Intentional: reset animation state when source text changes
  }, [text]);

  const flicker = useCallback(() => {
    const current = textRef.current;
    const count = Math.max(1, Math.floor(current.length * intensity));
    const result = current.split("").map((char) => ({ char, opacity: 1 }));

    for (let i = 0; i < count; i++) {
      const idx = Math.floor(Math.random() * current.length);
      if (result[idx].char !== " " && result[idx].char !== "\n") {
        result[idx].opacity = Math.random() < 0.5 ? 0 : 0.3;
      }
    }
    setChars(result);
  }, [intensity]);

  useEffect(() => {
    let isFlickered = false;

    intervalRef.current = setInterval(() => {
      if (isFlickered) {
        setChars(
          textRef.current.split("").map((char) => ({ char, opacity: 1 })),
        );
      } else {
        flicker();
      }
      isFlickered = !isFlickered;
    }, interval);

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    };
  }, [interval, flicker]);

  return (
    <span>
      {chars.map((c, i) =>
        c.opacity < 1 ? (
          <span
            key={i}
            style={{ opacity: c.opacity, transition: "opacity 0.2s ease" }}
          >
            {c.char}
          </span>
        ) : (
          c.char
        ),
      )}
    </span>
  );
}
