import { useEffect, useMemo, useState } from 'react';
import Particles, { initParticlesEngine } from '@tsparticles/react';
import { loadFull } from 'tsparticles';
import type { ISourceOptions } from '@tsparticles/engine';
import type { BackgroundAnimation } from '../theme/types';

interface BackgroundAnimationProps {
  animation: BackgroundAnimation;
  disabled?: boolean;
  /** The scroll container that the background should visually stick to */
  containerRef?: React.RefObject<HTMLElement | null>;
}

/** Particle configurations for each animation type */
const ANIMATION_CONFIGS: Record<BackgroundAnimation, ISourceOptions | null> = {
  none: null,
  
  stars: {
    fullScreen: false,
    background: { color: { value: 'transparent' } },
    fpsLimit: 60,
    particles: {
      number: { value: 80, density: { enable: true } },
      color: { value: ['#ffffff', '#fef3c7', '#bfdbfe', '#fbbf24'] },
      shape: { type: 'star' },
      opacity: { 
        value: { min: 0.2, max: 0.9 },
        animation: { enable: true, speed: 1, sync: false }
      },
      size: { value: { min: 2, max: 5 } },
      move: {
        enable: true,
        speed: 0.5,
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
    fpsLimit: 60,
    particles: {
      number: { value: 40, density: { enable: true } },
      color: { value: ['#67e8f9', '#a5f3fc', '#cffafe', '#22d3ee'] },
      shape: { type: 'circle' },
      opacity: { value: { min: 0.3, max: 0.7 } },
      size: { value: { min: 6, max: 16 } },
      move: {
        enable: true,
        speed: 1.5,
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
    fpsLimit: 60,
    particles: {
      number: { value: 35, density: { enable: true } },
      color: { value: ['#fef08a', '#fde047', '#facc15', '#fbbf24'] },
      shape: { type: 'circle' },
      opacity: { 
        value: { min: 0.3, max: 0.9 },
        animation: { enable: true, speed: 2, sync: false }
      },
      size: { value: { min: 3, max: 7 } },
      move: {
        enable: true,
        speed: 1,
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
    fpsLimit: 60,
    particles: {
      number: { value: 100, density: { enable: true } },
      color: { value: ['#ffffff', '#f0f9ff', '#e0f2fe'] },
      shape: { type: 'circle' },
      opacity: { value: { min: 0.5, max: 0.9 } },
      size: { value: { min: 2, max: 6 } },
      move: {
        enable: true,
        speed: 2,
        direction: 'bottom',
        random: false,
        straight: false,
        outModes: { default: 'out' },
        drift: 1.5,
      },
      wobble: {
        enable: true,
        distance: 12,
        speed: 4,
      },
      rotate: {
        animation: {
          enable: true,
          speed: 5,
          sync: false,
        },
      },
    },
    detectRetina: true,
  },
  
  rain: {
    fullScreen: false,
    background: { color: { value: 'transparent' } },
    fpsLimit: 60,
    particles: {
      number: { value: 100, density: { enable: true } },
      color: { value: ['#3b82f6', '#60a5fa', '#93c5fd'] },
      shape: { type: 'circle' },
      opacity: { value: { min: 0.6, max: 1.0 } },
      size: { value: { min: 1, max: 2 } },
      links: {
        enable: true,
        distance: 50,
        color: '#60a5fa',
        opacity: 0.4,
        width: 1,
      },
      move: {
        enable: true,
        speed: 8,
        direction: 'bottom',
        random: false,
        straight: true,
        outModes: { default: 'out' },
      },
    },
    detectRetina: true,
  },
  
  matrix: {
    fullScreen: false,
    background: { color: { value: 'transparent' } },
    fpsLimit: 60,
    particles: {
      number: { value: 250, density: { enable: true } },
      color: { value: '#00ff00' },
      shape: {
        type: 'text',
        options: {
          text: {
            value: ['0', '1'],
            font: 'monospace',
            style: '',
            weight: '400',
          }
        }
      },
      opacity: {
        value: { min: 0.1, max: 0.8 },
        animation: {
          enable: true,
          speed: 1,
          sync: false,
          startValue: 'min',
          destroy: 'none',
        },
      },
      size: { value: { min: 8, max: 14 } },
      move: {
        enable: true,
        speed: 1,
        direction: 'bottom',
        random: false,
        straight: true,
        outModes: { default: 'out' },
      },
    },
    detectRetina: true,
  },
  
  
  embers: {
    fullScreen: false,
    background: { color: { value: 'transparent' } },
    fpsLimit: 60,
    particles: {
      number: { value: 45, density: { enable: true } },
      color: { value: ['#ea580c', '#f97316', '#fb923c', '#fed7aa'] },
      shape: { type: 'circle' },
      opacity: {
        value: { min: 0.2, max: 0.9 },
        animation: {
          enable: true,
          speed: 3,
          sync: false,
          startValue: 'max',
          destroy: 'none',
        },
      },
      size: { value: { min: 2, max: 6 } },
      move: {
        enable: true,
        speed: { min: 0.5, max: 2 },
        direction: 'top',
        random: true,
        straight: false,
        outModes: { default: 'out' },
        gravity: {
          enable: true,
          acceleration: -0.3,
          maxSpeed: 3,
        },
      },
    },
    detectRetina: true,
  },
  
  hyperspace: {
    fullScreen: false,
    background: { color: { value: 'transparent' } },
    fpsLimit: 60,
    particles: {
      number: {
        value: 200,
        density: {
          enable: true,
        },
      },
      color: {
        value: ['#ffffff', '#e0e7ff', '#c7d2fe', '#a5b4fc', '#818cf8'],
      },
      shape: {
        type: 'circle',
      },
      opacity: {
        value: { min: 0.3, max: 1 },
      },
      size: {
        value: {
          min: 1,
          max: 4,
        },
      },
      move: {
        enable: true,
        speed: {
          min: 2,
          max: 15,
        },
        direction: 'outside',
        straight: true,
        outModes: {
          default: 'destroy',
        },
      },
    },
    emitters: {
      position: {
        x: 50,
        y: 50,
      },
      size: {
        width: 100,
        height: 100,
      },
      rate: {
        quantity: 15,
        delay: 0.05,
      },
    },
    detectRetina: true,
  },
  
};

// Initialize engine once globally
let engineInitialized = false;
let engineInitPromise: Promise<void> | null = null;

export function BackgroundAnimation({ animation, disabled = false, containerRef }: BackgroundAnimationProps) {
  const [init, setInit] = useState(engineInitialized);
  const [scrollTop, setScrollTop] = useState(0);
  
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
        await loadFull(engine);
      }).then(() => {
        engineInitialized = true;
      });
    }
    
    engineInitPromise.then(() => setInit(true));
  }, []);

  // Keep the particles layer visually fixed within the scroll container.
  // Without this, an absolutely positioned layer would remain at scrollTop=0 and disappear once you scroll.
  useEffect(() => {
    const el = containerRef?.current;
    if (!el || typeof window === 'undefined') {
      setScrollTop(0);
      return;
    }

    let raf = 0;
    let lastScrollTime = 0;

    const update = () => {
      raf = 0;
      const now = performance.now();
      // Throttle updates to prevent excessive re-renders during fast scrolling
      if (now - lastScrollTime > 16) { // ~60fps
        setScrollTop(el.scrollTop);
        lastScrollTime = now;
      }
    };

    const onScroll = () => {
      if (raf) return;
      raf = window.requestAnimationFrame(update);
    };

    // Initialize and subscribe
    setScrollTop(el.scrollTop);
    el.addEventListener('scroll', onScroll, { passive: true });

    return () => {
      if (raf) window.cancelAnimationFrame(raf);
      el.removeEventListener('scroll', onScroll);
    };
  }, [containerRef]);
  
  // Don't render if not initialized, disabled, no animation, or user prefers reduced motion
  if (!init || disabled || animation === 'none' || prefersReducedMotion) {
    return null;
  }
  
  const config = ANIMATION_CONFIGS[animation];
  if (!config) return null;
  
  return (
    <div
      aria-hidden="true"
      style={{
        position: 'absolute',
        inset: 0,
        pointerEvents: 'none',
        zIndex: 0,
        overflow: 'hidden',
        transform: `translateY(${scrollTop}px)`,
        willChange: 'transform',
      }}
    >
      <Particles
        id="game-bg-particles"
        options={config}
        style={{ width: '100%', height: '100%' }}
      />
    </div>
  );
}
