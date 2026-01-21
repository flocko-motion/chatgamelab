import { useEffect, useMemo, useState } from 'react';
import Particles, { initParticlesEngine } from '@tsparticles/react';
import { loadSlim } from '@tsparticles/slim';
import type { ISourceOptions } from '@tsparticles/engine';
import type { BackgroundAnimation } from '../theme/types';

interface BackgroundAnimationProps {
  animation: BackgroundAnimation;
  disabled?: boolean;
}

/** Particle configurations for each animation type */
const ANIMATION_CONFIGS: Record<BackgroundAnimation, ISourceOptions | null> = {
  none: null,
  
  stars: {
    fullScreen: false,
    background: { color: { value: 'transparent' } },
    fpsLimit: 30,
    particles: {
      number: { value: 50, density: { enable: true } },
      color: { value: ['#ffffff', '#fef3c7', '#bfdbfe'] },
      shape: { type: 'circle' },
      opacity: { 
        value: { min: 0.3, max: 0.8 },
        animation: { enable: true, speed: 0.5, sync: false }
      },
      size: { value: { min: 1, max: 3 } },
      move: {
        enable: true,
        speed: 0.3,
        direction: 'none',
        random: true,
        straight: false,
        outModes: { default: 'out' },
      },
      twinkle: {
        particles: { enable: true, frequency: 0.05, opacity: 1 }
      }
    },
    detectRetina: true,
  },
  
  bubbles: {
    fullScreen: false,
    background: { color: { value: 'transparent' } },
    fpsLimit: 30,
    particles: {
      number: { value: 30, density: { enable: true } },
      color: { value: ['#67e8f9', '#a5f3fc', '#cffafe'] },
      shape: { type: 'circle' },
      opacity: { value: { min: 0.2, max: 0.5 } },
      size: { value: { min: 4, max: 12 } },
      move: {
        enable: true,
        speed: 1,
        direction: 'top',
        random: true,
        straight: false,
        outModes: { default: 'out', bottom: 'out', top: 'out' },
      },
    },
    detectRetina: true,
  },
  
  fireflies: {
    fullScreen: false,
    background: { color: { value: 'transparent' } },
    fpsLimit: 30,
    particles: {
      number: { value: 25, density: { enable: true } },
      color: { value: ['#fef08a', '#fde047', '#facc15'] },
      shape: { type: 'circle' },
      opacity: { 
        value: { min: 0.2, max: 0.8 },
        animation: { enable: true, speed: 1, sync: false }
      },
      size: { value: { min: 2, max: 5 } },
      move: {
        enable: true,
        speed: 0.8,
        direction: 'none',
        random: true,
        straight: false,
        outModes: { default: 'bounce' },
      },
    },
    detectRetina: true,
  },
  
  snow: {
    fullScreen: false,
    background: { color: { value: 'transparent' } },
    fpsLimit: 30,
    particles: {
      number: { value: 60, density: { enable: true } },
      color: { value: '#ffffff' },
      shape: { type: 'circle' },
      opacity: { value: { min: 0.4, max: 0.8 } },
      size: { value: { min: 2, max: 5 } },
      move: {
        enable: true,
        speed: 1.5,
        direction: 'bottom',
        random: false,
        straight: false,
        outModes: { default: 'out' },
        drift: 1,
      },
      wobble: {
        enable: true,
        distance: 10,
        speed: 5,
      },
    },
    detectRetina: true,
  },
  
  rain: {
    fullScreen: false,
    background: { color: { value: 'transparent' } },
    fpsLimit: 30,
    particles: {
      number: { value: 80, density: { enable: true } },
      color: { value: ['#94a3b8', '#cbd5e1'] },
      shape: { type: 'line' },
      opacity: { value: { min: 0.3, max: 0.6 } },
      size: { value: { min: 1, max: 2 } },
      move: {
        enable: true,
        speed: 15,
        direction: 'bottom',
        random: false,
        straight: true,
        outModes: { default: 'out' },
      },
      rotate: {
        value: 70,
        direction: 'clockwise',
        animation: { enable: false },
      },
    },
    detectRetina: true,
  },
  
  matrix: {
    fullScreen: false,
    background: { color: { value: 'transparent' } },
    fpsLimit: 30,
    particles: {
      number: { value: 50, density: { enable: true } },
      color: { value: '#00ff00' },
      shape: { 
        type: 'char',
        options: {
          char: {
            value: ['0', '1', '2', '3', '4', '5', '6', '7', '8', '9', 'A', 'B', 'C', 'D', 'E', 'F'],
            font: 'monospace',
            style: '',
            weight: '400',
          }
        }
      },
      opacity: { 
        value: { min: 0.1, max: 0.8 },
        animation: { enable: true, speed: 1, sync: false }
      },
      size: { value: { min: 8, max: 14 } },
      move: {
        enable: true,
        speed: 3,
        direction: 'bottom',
        random: false,
        straight: true,
        outModes: { default: 'out' },
      },
    },
    detectRetina: true,
  },
};

// Initialize engine once globally
let engineInitialized = false;
let engineInitPromise: Promise<void> | null = null;

export function BackgroundAnimation({ animation, disabled = false }: BackgroundAnimationProps) {
  const [init, setInit] = useState(engineInitialized);
  
  // Respect prefers-reduced-motion
  const prefersReducedMotion = useMemo(() => {
    if (typeof window === 'undefined') return false;
    return window.matchMedia('(prefers-reduced-motion: reduce)').matches;
  }, []);
  
  // Initialize tsParticles engine once (pattern from official tsParticles docs)
  /* eslint-disable @eslint-react/hooks-extra/no-direct-set-state-in-use-effect -- Official tsParticles initialization pattern */
  useEffect(() => {
    if (engineInitialized) {
      setInit(true);
      return;
    }
    
    if (!engineInitPromise) {
      engineInitPromise = initParticlesEngine(async (engine) => {
        await loadSlim(engine);
      }).then(() => {
        engineInitialized = true;
      });
    }
    
    engineInitPromise.then(() => setInit(true));
  }, []);
  
  // Don't render if not initialized, disabled, no animation, or user prefers reduced motion
  if (!init || disabled || animation === 'none' || prefersReducedMotion) {
    return null;
  }
  
  const config = ANIMATION_CONFIGS[animation];
  if (!config) return null;
  
  return (
    <Particles
      id="game-bg-particles"
      options={config}
      style={{
        position: 'absolute',
        top: 0,
        left: 0,
        width: '100%',
        height: '100%',
        pointerEvents: 'none',
        zIndex: 0,
      }}
    />
  );
}
