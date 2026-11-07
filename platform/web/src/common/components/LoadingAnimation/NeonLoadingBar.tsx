import React from 'react';
import { Box, useMantineTheme } from '@mantine/core';
import styles from './NeonLoadingBar.module.css';

export interface NeonLoadingBarProps {
  visible: boolean;
  color?: string;
  minVisibleMs?: number;
}

export function NeonLoadingBar({ visible, color, minVisibleMs = 250 }: NeonLoadingBarProps) {
  const theme = useMantineTheme();
  const glowColor = color || theme.colors.violet[5];
  const rgbColor = hexToRgb(glowColor);

  const [shownVisible, setShownVisible] = React.useState(false);
  const showStartedAtRef = React.useRef<number | null>(null);
  const hideTimerRef = React.useRef<number | null>(null);

  React.useEffect(() => {
    if (visible) {
      if (hideTimerRef.current) {
        window.clearTimeout(hideTimerRef.current);
        hideTimerRef.current = null;
      }
      if (!shownVisible) {
        showStartedAtRef.current = Date.now();
        setShownVisible(true);
      }
      return;
    }

    if (!shownVisible) return;

    const startedAt = showStartedAtRef.current ?? Date.now();
    const elapsed = Date.now() - startedAt;
    const remaining = Math.max(0, minVisibleMs - elapsed);

    if (hideTimerRef.current) {
      window.clearTimeout(hideTimerRef.current);
    }

    hideTimerRef.current = window.setTimeout(() => {
      showStartedAtRef.current = null;
      setShownVisible(false);
      hideTimerRef.current = null;
    }, remaining);
  }, [minVisibleMs, shownVisible, visible]);

  return (
    <Box className={styles.container}>
      <Box
        className={`${styles.bar} ${shownVisible ? styles.barVisible : ''}`}
        style={{
          '--glow-color-rgb': rgbColor,
        } as React.CSSProperties}
      >
        <div className={styles.gradient} />
        <div className={styles.spark} />
      </Box>
    </Box>
  );
}

function hexToRgb(hex: string): string {
  const shorthandRegex = /^#?([a-f\d])([a-f\d])([a-f\d])$/i;
  const fullHex = hex.replace(shorthandRegex, (_, r, g, b) => r + r + g + g + b + b);
  const result = /^#?([a-f\d]{2})([a-f\d]{2})([a-f\d]{2})$/i.exec(fullHex);

  if (result) {
    return `${parseInt(result[1], 16)}, ${parseInt(result[2], 16)}, ${parseInt(result[3], 16)}`;
  }

  return '124, 58, 237';
}
