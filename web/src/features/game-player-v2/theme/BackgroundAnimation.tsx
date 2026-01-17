/**
 * Background Animation Components
 * 
 * Provides animated backgrounds for the game player scene area.
 * All animations are purely visual and non-destructive.
 */

import { useMemo } from 'react';
import type { BackgroundAnimation as AnimationType } from './types';
import styles from './BackgroundAnimation.module.css';

interface BackgroundAnimationProps {
  animation: AnimationType;
  className?: string;
}

export function BackgroundAnimation({ animation, className }: BackgroundAnimationProps) {
  const animationClass = useMemo(() => {
    switch (animation) {
      case 'stars': return styles.stars;
      case 'rain': return styles.rain;
      case 'fog': return styles.fog;
      case 'particles': return styles.particles;
      case 'scanlines': return styles.scanlines;
      default: return '';
    }
  }, [animation]);
  
  if (animation === 'none') {
    return null;
  }
  
  return (
    <div 
      className={`${styles.animationLayer} ${animationClass} ${className || ''}`}
      aria-hidden="true"
    />
  );
}
