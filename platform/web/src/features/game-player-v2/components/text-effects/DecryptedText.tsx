/**
 * DecryptedText effect for AI messages.
 * Adapted from ReactBits (https://reactbits.dev/text-animations/decrypted-text)
 *
 * Shows scrambled characters that progressively reveal the real text.
 * Designed for the hacker/terminal theme.
 */

import { useEffect, useState, useRef, useCallback } from "react";

const CHARS =
  "ABCDEFGHIJKLMNOPQRSTUVWXYZabcdefghijklmnopqrstuvwxyz0123456789!@#$%^&*()_+-=[]{}|;:,.<>?";

interface DecryptedTextProps {
  text: string;
  /** ms between each reveal step */
  speed?: number;
}

export function DecryptedText({ text, speed = 20 }: DecryptedTextProps) {
  const [displayText, setDisplayText] = useState(text);
  const revealedCountRef = useRef(0);
  const intervalRef = useRef<ReturnType<typeof setInterval> | null>(null);

  const scramble = useCallback((original: string, revealedCount: number) => {
    const chars = CHARS.split("");
    return original
      .split("")
      .map((char, i) => {
        if (char === " " || char === "\n") return char;
        if (i < revealedCount) return original[i];
        return chars[Math.floor(Math.random() * chars.length)];
      })
      .join("");
  }, []);

  useEffect(() => {
    // When text grows (streaming), animate the new characters
    const prevRevealed = revealedCountRef.current;

    // If text shrunk or is empty, reset
    if (text.length <= prevRevealed) {
      revealedCountRef.current = text.length;
      setDisplayText(text);
      return;
    }

    // Clear any running animation
    if (intervalRef.current) {
      clearInterval(intervalRef.current);
    }

    let currentRevealed = prevRevealed;

    intervalRef.current = setInterval(() => {
      if (currentRevealed >= text.length) {
        if (intervalRef.current) clearInterval(intervalRef.current);
        intervalRef.current = null;
        revealedCountRef.current = text.length;
        setDisplayText(text);
        return;
      }

      // Reveal next characters (2 per tick for snappier feel)
      currentRevealed = Math.min(currentRevealed + 2, text.length);
      revealedCountRef.current = currentRevealed;
      setDisplayText(scramble(text, currentRevealed));
    }, speed);

    // Show scrambled version immediately
    setDisplayText(scramble(text, currentRevealed));

    return () => {
      if (intervalRef.current) {
        clearInterval(intervalRef.current);
        intervalRef.current = null;
      }
    };
  }, [text, speed, scramble]);

  return <span>{displayText}</span>;
}
