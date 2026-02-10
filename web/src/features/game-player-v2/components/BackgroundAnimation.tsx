import { memo, useEffect, useMemo, useRef, useState } from "react";
import Particles, { initParticlesEngine } from "@tsparticles/react";
import { loadFull } from "tsparticles";
import type { ISourceOptions } from "@tsparticles/engine";
import type { BackgroundAnimation as BackgroundAnimationType } from "../theme/types";

interface BackgroundAnimationProps {
  animation: BackgroundAnimationType;
  disabled?: boolean;
}

/** Particle configurations for each animation type */
const ANIMATION_CONFIGS: Record<
  BackgroundAnimationType,
  ISourceOptions | null
> = {
  none: null,

  stars: {
    fullScreen: false,
    background: { color: { value: "transparent" } },
    fpsLimit: 60,
    particles: {
      number: { value: 180, density: { enable: true } },
      color: { value: ["#ffffff", "#f8fafc", "#e2e8f0", "#f1f5f9"] },
      shape: { type: "circle" },
      opacity: {
        value: { min: 0.2, max: 0.9 },
        animation: { enable: true, speed: 0.8, sync: false },
      },
      size: {
        value: { min: 0.5, max: 2.5 },
        animation: {
          enable: true,
          speed: 1,
          sync: false,
          startValue: "random",
        },
      },
    },
    detectRetina: true,
  },

  bubbles: {
    fullScreen: false,
    background: { color: { value: "transparent" } },
    fpsLimit: 60,
    particles: {
      number: { value: 40, density: { enable: true } },
      color: { value: ["#67e8f9", "#a5f3fc", "#cffafe", "#22d3ee"] },
      shape: { type: "circle" },
      opacity: { value: { min: 0.3, max: 0.7 } },
      size: { value: { min: 6, max: 16 } },
      move: {
        enable: true,
        speed: 1.5,
        direction: "top",
        random: true,
        straight: false,
        outModes: { default: "out", bottom: "out", top: "out" },
      },
    },
    detectRetina: true,
  },

  fireflies: {
    fullScreen: false,
    background: { color: { value: "transparent" } },
    fpsLimit: 60,
    particles: {
      number: { value: 35, density: { enable: true } },
      color: { value: ["#fef08a", "#fde047", "#facc15", "#fbbf24"] },
      shape: { type: "circle" },
      opacity: {
        value: { min: 0.3, max: 0.9 },
        animation: { enable: true, speed: 2, sync: false },
      },
      size: { value: { min: 3, max: 7 } },
      move: {
        enable: true,
        speed: 1,
        direction: "none",
        random: true,
        straight: false,
        outModes: { default: "bounce" },
      },
    },
    detectRetina: true,
  },

  snow: {
    fullScreen: false,
    background: { color: { value: "transparent" } },
    fpsLimit: 60,
    particles: {
      number: { value: 100, density: { enable: true } },
      color: { value: ["#ffffff", "#f0f9ff", "#e0f2fe"] },
      shape: { type: "circle" },
      opacity: { value: { min: 0.5, max: 0.9 } },
      size: { value: { min: 2, max: 6 } },
      move: {
        enable: true,
        speed: 1.5,
        direction: "bottom",
        straight: true,
        outModes: { default: "out" },
        gravity: {
          enable: false,
        },
      },
    },
    detectRetina: true,
  },

  bits: {
    fullScreen: false,
    background: { color: { value: "transparent" } },
    fpsLimit: 60,
    particles: {
      number: { value: 250, density: { enable: true } },
      color: { value: "#00ff00" },
      shape: {
        type: "text",
        options: {
          text: {
            value: ["0", "1"],
            font: "monospace",
            style: "",
            weight: "400",
          },
        },
      },
      opacity: {
        value: { min: 0.1, max: 0.8 },
        animation: {
          enable: true,
          speed: 1,
          sync: false,
          startValue: "min",
          destroy: "none",
        },
      },
      size: { value: { min: 8, max: 14 } },
      move: {
        enable: true,
        speed: 1,
        direction: "bottom",
        random: false,
        straight: true,
        outModes: { default: "out" },
      },
    },
    detectRetina: true,
  },

  matrixRain: null, // Canvas-based animation (real Matrix digital rain)

  embers: {
    fullScreen: false,
    background: { color: { value: "transparent" } },
    fpsLimit: 60,
    particles: {
      number: { value: 45, density: { enable: true } },
      color: { value: ["#ea580c", "#f97316", "#fb923c", "#fed7aa"] },
      shape: { type: "circle" },
      opacity: {
        value: { min: 0.2, max: 0.9 },
        animation: {
          enable: true,
          speed: 3,
          sync: false,
          startValue: "max",
          destroy: "none",
        },
      },
      size: { value: { min: 2, max: 6 } },
      move: {
        enable: true,
        speed: { min: 0.5, max: 1.5 },
        direction: "top",
        random: true,
        straight: false,
        outModes: { default: "out" },
      },
    },
    detectRetina: true,
  },

  hyperspace: {
    fullScreen: false,
    background: { color: { value: "transparent" } },
    fpsLimit: 60,
    particles: {
      number: {
        value: 200,
        density: {
          enable: true,
        },
      },
      color: {
        value: ["#ffffff", "#e0e7ff", "#c7d2fe", "#a5b4fc", "#818cf8"],
      },
      shape: {
        type: "circle",
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
        direction: "outside",
        straight: true,
        outModes: {
          default: "destroy",
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

  sparkles: {
    fullScreen: false,
    background: { color: { value: "transparent" } },
    fpsLimit: 60,
    particles: {
      number: { value: 50, density: { enable: true } },
      color: { value: ["#10b981", "#34d399", "#6ee7b7", "#a7f3d0", "#d1fae5"] },
      shape: { type: "star" },
      opacity: {
        value: { min: 0.3, max: 1 },
        animation: { enable: true, speed: 2, sync: false },
      },
      size: {
        value: { min: 3, max: 8 },
        animation: { enable: true, speed: 3, sync: false },
      },
      move: {
        enable: true,
        speed: 0.8,
        direction: "none",
        random: true,
        straight: false,
        outModes: { default: "bounce" },
      },
      rotate: {
        value: { min: 0, max: 360 },
        animation: { enable: true, speed: 5, sync: false },
      },
    },
    detectRetina: true,
  },

  hearts: {
    fullScreen: false,
    background: { color: { value: "transparent" } },
    fpsLimit: 60,
    particles: {
      number: { value: 25, density: { enable: true } },
      color: { value: ["#f43f5e", "#fb7185", "#fda4af", "#fecdd3"] },
      shape: {
        type: "text",
        options: {
          text: {
            value: ["♥", "♡", "❤"],
            font: "serif",
            style: "",
            weight: "400",
          },
        },
      },
      opacity: {
        value: { min: 0.4, max: 0.9 },
        animation: { enable: true, speed: 1, sync: false },
      },
      size: { value: { min: 10, max: 20 } },
      move: {
        enable: true,
        speed: 1,
        direction: "top",
        random: true,
        straight: false,
        outModes: { default: "out" },
      },
      wobble: {
        enable: true,
        distance: 10,
        speed: 3,
      },
    },
    detectRetina: true,
  },

  glitch: {
    fullScreen: false,
    background: { color: { value: "transparent" } },
    fpsLimit: 60,
    particles: {
      number: { value: 80, density: { enable: true } },
      color: { value: ["#ff0000", "#00ff00", "#0000ff", "#ff00ff", "#00ffff"] },
      shape: { type: "square" },
      opacity: {
        value: { min: 0.2, max: 0.8 },
        animation: { enable: true, speed: 8, sync: false },
      },
      size: { value: { min: 1, max: 15 } },
      move: {
        enable: true,
        speed: { min: 1, max: 5 },
        direction: "none",
        random: true,
        straight: true,
        outModes: { default: "out" },
      },
      twinkle: {
        particles: { enable: true, frequency: 0.2, opacity: 1 },
      },
    },
    detectRetina: true,
  },

  circuits: {
    fullScreen: false,
    background: { color: { value: "transparent" } },
    fpsLimit: 60,
    particles: {
      number: { value: 60, density: { enable: true } },
      color: { value: ["#06b6d4", "#22d3ee", "#67e8f9", "#a5f3fc"] },
      shape: { type: "circle" },
      opacity: { value: { min: 0.4, max: 0.9 } },
      size: { value: { min: 2, max: 4 } },
      links: {
        enable: true,
        distance: 120,
        color: "#22d3ee",
        opacity: 0.5,
        width: 1,
      },
      move: {
        enable: true,
        speed: 1.5,
        direction: "none",
        random: false,
        straight: false,
        outModes: { default: "bounce" },
      },
    },
    detectRetina: true,
  },

  leaves: {
    fullScreen: false,
    background: { color: { value: "transparent" } },
    fpsLimit: 60,
    particles: {
      number: { value: 30, density: { enable: true } },
      color: {
        value: [
          "#16a34a",
          "#22c55e",
          "#4ade80",
          "#86efac",
          "#a16207",
          "#ca8a04",
        ],
      },
      shape: {
        type: "text",
        options: {
          text: {
            value: ["🍃", "🍂", "🌿", "☘️"],
            font: "serif",
            style: "",
            weight: "400",
          },
        },
      },
      opacity: { value: { min: 0.6, max: 0.9 } },
      size: { value: { min: 14, max: 20 } },
      move: {
        enable: true,
        speed: { min: 0.5, max: 1.5 },
        direction: "bottom-right",
        straight: true,
        outModes: { default: "out" },
      },
    },
    detectRetina: true,
  },

  geometric: {
    fullScreen: false,
    background: { color: { value: "transparent" } },
    fpsLimit: 60,
    particles: {
      number: { value: 40, density: { enable: true } },
      color: {
        value: [
          "#8b5cf6",
          "#a78bfa",
          "#c4b5fd",
          "#06b6d4",
          "#22d3ee",
          "#f472b6",
        ],
      },
      shape: {
        type: ["triangle", "square", "polygon"],
        options: {
          polygon: { sides: 6 },
        },
      },
      opacity: {
        value: { min: 0.2, max: 0.6 },
        animation: { enable: true, speed: 1, sync: false },
      },
      size: {
        value: { min: 8, max: 25 },
        animation: { enable: true, speed: 2, sync: false },
      },
      move: {
        enable: true,
        speed: 0.8,
        direction: "none",
        random: true,
        straight: false,
        outModes: { default: "bounce" },
      },
      rotate: {
        value: { min: 0, max: 360 },
        animation: { enable: true, speed: 3, sync: false },
      },
    },
    detectRetina: true,
  },

  confetti: {
    fullScreen: false,
    background: { color: { value: "transparent" } },
    fpsLimit: 60,
    particles: {
      number: { value: 40, density: { enable: true } },
      color: {
        value: [
          "#f472b6",
          "#fb923c",
          "#facc15",
          "#4ade80",
          "#60a5fa",
          "#a78bfa",
          "#f87171",
        ],
      },
      shape: {
        type: ["circle", "square", "triangle"],
      },
      opacity: { value: { min: 0.7, max: 1 } },
      size: { value: { min: 5, max: 12 } },
      move: {
        enable: true,
        speed: 1.5,
        direction: "bottom",
        straight: true,
        outModes: { default: "out" },
        gravity: {
          enable: false,
        },
      },
      rotate: {
        value: { min: 0, max: 360 },
        animation: { enable: true, speed: 5, sync: false },
      },
    },
    detectRetina: true,
  },

  confettiExplosion: {
    fullScreen: false,
    background: { color: { value: "transparent" } },
    fpsLimit: 60,
    emitters: [
      {
        position: { x: 20, y: 30 },
        rate: { quantity: 12, delay: 0.1 },
        size: { width: 5, height: 5 },
        life: { count: 0, duration: 0.15, delay: 3 },
      },
      {
        position: { x: 75, y: 20 },
        rate: { quantity: 12, delay: 0.1 },
        size: { width: 5, height: 5 },
        life: { count: 0, duration: 0.15, delay: 4.5 },
      },
      {
        position: { x: 50, y: 50 },
        rate: { quantity: 12, delay: 0.1 },
        size: { width: 5, height: 5 },
        life: { count: 0, duration: 0.15, delay: 5.5 },
      },
      {
        position: { x: 35, y: 70 },
        rate: { quantity: 12, delay: 0.1 },
        size: { width: 5, height: 5 },
        life: { count: 0, duration: 0.15, delay: 7 },
      },
      {
        position: { x: 85, y: 60 },
        rate: { quantity: 12, delay: 0.1 },
        size: { width: 5, height: 5 },
        life: { count: 0, duration: 0.15, delay: 8.5 },
      },
      {
        position: { x: 10, y: 55 },
        rate: { quantity: 12, delay: 0.1 },
        size: { width: 5, height: 5 },
        life: { count: 0, duration: 0.15, delay: 10 },
      },
      {
        position: { x: 60, y: 35 },
        rate: { quantity: 12, delay: 0.1 },
        size: { width: 5, height: 5 },
        life: { count: 0, duration: 0.15, delay: 11.5 },
      },
      {
        position: { x: 40, y: 15 },
        rate: { quantity: 12, delay: 0.1 },
        size: { width: 5, height: 5 },
        life: { count: 0, duration: 0.15, delay: 13 },
      },
      {
        position: { x: 90, y: 40 },
        rate: { quantity: 12, delay: 0.1 },
        size: { width: 5, height: 5 },
        life: { count: 0, duration: 0.15, delay: 14.5 },
      },
      {
        position: { x: 25, y: 80 },
        rate: { quantity: 12, delay: 0.1 },
        size: { width: 5, height: 5 },
        life: { count: 0, duration: 0.15, delay: 16 },
      },
    ],
    particles: {
      number: { value: 0 },
      color: {
        value: [
          "#f472b6",
          "#fb923c",
          "#facc15",
          "#4ade80",
          "#60a5fa",
          "#a78bfa",
          "#f87171",
          "#e879f9",
          "#34d399",
        ],
      },
      shape: {
        type: ["circle", "square", "triangle"],
      },
      opacity: {
        value: { min: 0.6, max: 1 },
        animation: {
          enable: true,
          speed: 0.5,
          sync: false,
          startValue: "max",
          destroy: "min",
        },
      },
      size: { value: { min: 4, max: 10 } },
      move: {
        enable: true,
        speed: { min: 8, max: 20 },
        direction: "none",
        random: true,
        straight: false,
        outModes: { default: "destroy" },
        gravity: {
          enable: true,
          acceleration: 5,
        },
      },
      rotate: {
        value: { min: 0, max: 360 },
        animation: { enable: true, speed: 15, sync: false },
      },
      tilt: {
        enable: true,
        value: { min: 0, max: 360 },
        animation: { enable: true, speed: 10, sync: false },
      },
      life: {
        count: 1,
        duration: { value: { min: 2, max: 4 } },
      },
    },
    detectRetina: true,
  },

  waves: null, // CSS-based animation (tsparticles can't do wave motion)
  sun: null, // CSS-based animation (single glowing orb arc)
  tumbleweed: null, // CSS-based animation (rolling bushes along bottom)

  glowworm: {
    fullScreen: false,
    background: { color: { value: "transparent" } },
    fpsLimit: 60,
    particles: {
      number: { value: 60, density: { enable: true } },
      color: { value: ["#4ade80", "#86efac", "#22c55e", "#bbf7d0"] },
      shape: { type: "circle" },
      opacity: {
        value: { min: 0.15, max: 0.9 },
        animation: {
          enable: true,
          speed: 0.8,
          sync: false,
          startValue: "random",
        },
      },
      size: {
        value: { min: 2, max: 7 },
        animation: {
          enable: true,
          speed: 1,
          sync: false,
          startValue: "random",
        },
      },
      move: {
        enable: true,
        speed: 0.4,
        direction: "none",
        random: true,
        straight: false,
        outModes: { default: "bounce" },
      },
      shadow: {
        enable: true,
        color: "#4ade80",
        blur: 10,
      },
    },
    detectRetina: true,
  },
};

/** CSS-based waves background using SVG sine-wave paths */
const WAVE_KEYFRAMES = `
@keyframes waveScroll {
  0% { transform: translateX(0); }
  100% { transform: translateX(-50%); }
}
`;

let waveKeyframesInjected = false;
function ensureWaveKeyframes() {
  if (waveKeyframesInjected) return;
  if (typeof document === "undefined") return;
  const style = document.createElement("style");
  style.textContent = WAVE_KEYFRAMES;
  document.head.appendChild(style);
  waveKeyframesInjected = true;
}

/** Build a closed SVG path: sine wave on top, filled down to bottom */
function waveFilledPath(
  width: number,
  amplitude: number,
  frequency: number,
  yOffset: number,
  height: number,
): string {
  const points: string[] = [];
  const steps = 200;
  for (let i = 0; i <= steps; i++) {
    const x = (i / steps) * width;
    const y =
      yOffset + Math.sin((i / steps) * Math.PI * 2 * frequency) * amplitude;
    points.push(`${i === 0 ? "M" : "L"}${x.toFixed(1)},${y.toFixed(1)}`);
  }
  // Close path down to bottom-right, bottom-left, back to start
  points.push(`L${width},${height} L0,${height} Z`);
  return points.join(" ");
}

function WavesBackground() {
  ensureWaveKeyframes();

  // Waves confined to bottom 1/3 of the container.
  // yOffset is relative to the SVG viewBox height (100).
  // [amplitude, frequency, yOffset, strokeColor, fillColor, fillOpacity, strokeOpacity, duration, strokeWidth]
  const layers: [
    number,
    number,
    number,
    string,
    string,
    number,
    number,
    string,
    number,
  ][] = [
    [6, 2, 15, "#3b82f6", "#3b82f6", 0.06, 0.3, "14s", 1.5],
    [4, 3, 35, "#60a5fa", "#60a5fa", 0.05, 0.25, "11s", 1],
    [8, 1.5, 50, "#2563eb", "#2563eb", 0.08, 0.2, "16s", 2],
    [5, 2.5, 65, "#93c5fd", "#93c5fd", 0.04, 0.2, "12s", 1],
    [3, 4, 80, "#60a5fa", "#60a5fa", 0.03, 0.15, "9s", 0.8],
  ];

  const svgW = 800;
  const svgH = 100;

  return (
    <div
      aria-hidden="true"
      style={{
        position: "absolute",
        left: 0,
        right: 0,
        bottom: 0,
        height: "33%",
        pointerEvents: "none",
        zIndex: 0,
        overflow: "hidden",
      }}
    >
      {layers.map(
        (
          [amp, freq, yPct, strokeCol, fillCol, fillOp, strokeOp, dur, sw],
          i,
        ) => (
          <div
            key={i}
            style={{
              position: "absolute",
              inset: 0,
              animation: `waveScroll ${dur} linear infinite`,
            }}
          >
            <svg
              viewBox={`0 0 ${svgW * 2} ${svgH}`}
              preserveAspectRatio="none"
              style={{
                position: "absolute",
                top: 0,
                left: 0,
                width: "200%",
                height: "100%",
              }}
            >
              <path
                d={waveFilledPath(svgW * 2, amp, freq * 2, yPct, svgH)}
                fill={fillCol}
                fillOpacity={fillOp}
                stroke={strokeCol}
                strokeOpacity={strokeOp}
                strokeWidth={sw}
              />
            </svg>
          </div>
        ),
      )}
    </div>
  );
}

/** CSS-based sun background — glowing orb arcing slowly across the sky */
const SUN_KEYFRAMES = `
@keyframes sunArc {
  0% {
    transform: translate(-10%, 60%) scale(0.9);
    opacity: 0.4;
  }
  15% {
    transform: translate(10%, 20%) scale(1);
    opacity: 0.7;
  }
  50% {
    transform: translate(50%, 5%) scale(1.1);
    opacity: 0.9;
  }
  85% {
    transform: translate(90%, 20%) scale(1);
    opacity: 0.7;
  }
  100% {
    transform: translate(110%, 60%) scale(0.9);
    opacity: 0.4;
  }
}
`;

let sunKeyframesInjected = false;
function ensureSunKeyframes() {
  if (sunKeyframesInjected) return;
  if (typeof document === "undefined") return;
  const style = document.createElement("style");
  style.textContent = SUN_KEYFRAMES;
  document.head.appendChild(style);
  sunKeyframesInjected = true;
}

function SunBackground() {
  ensureSunKeyframes();

  return (
    <div
      aria-hidden="true"
      style={{
        position: "absolute",
        inset: 0,
        pointerEvents: "none",
        zIndex: 0,
        overflow: "hidden",
      }}
    >
      <div
        style={{
          position: "absolute",
          top: "5%",
          left: 0,
          width: 100,
          height: 100,
          borderRadius: "50%",
          background:
            "radial-gradient(circle, rgba(253, 224, 71, 0.9) 0%, rgba(250, 204, 21, 0.6) 30%, rgba(234, 179, 8, 0.2) 60%, transparent 75%)",
          boxShadow:
            "0 0 40px rgba(253, 224, 71, 0.5), 0 0 80px rgba(250, 204, 21, 0.3), 0 0 120px rgba(234, 179, 8, 0.15)",
          animation: "sunArc 60s ease-in-out infinite",
        }}
      />
    </div>
  );
}

/**
 * CSS-based tumbleweed + dust background for the western theme.
 *
 * Approach:
 * - Tumbleweeds use a SINGLE `transform`-based animation per weed that combines
 *   translateX (drift), translateY (bounce), and rotate (spin). This is GPU-
 *   accelerated and avoids the "stuck at left" issues of animating `left`.
 * - Wind/dust is a scrolling semi-transparent gradient overlay that creates a
 *   haze effect, like dust blowing across the scene.
 */
const TUMBLEWEED_KEYFRAMES = `
@keyframes tw1 {
  0%   { transform: translateX(-10vw) translateY(0px)   rotate(0deg); }
  8%   { transform: translateX(0vw)   translateY(-35px)  rotate(60deg); }
  16%  { transform: translateX(10vw)  translateY(0px)    rotate(130deg); }
  28%  { transform: translateX(22vw)  translateY(-50px)  rotate(230deg); }
  38%  { transform: translateX(34vw)  translateY(0px)    rotate(340deg); }
  50%  { transform: translateX(48vw)  translateY(-25px)  rotate(450deg); }
  60%  { transform: translateX(58vw)  translateY(0px)    rotate(530deg); }
  72%  { transform: translateX(70vw)  translateY(-40px)  rotate(640deg); }
  84%  { transform: translateX(84vw)  translateY(0px)    rotate(740deg); }
  92%  { transform: translateX(96vw)  translateY(-15px)  rotate(810deg); }
  100% { transform: translateX(110vw) translateY(0px)    rotate(900deg); }
}
@keyframes tw2 {
  0%   { transform: translateX(-10vw) translateY(0px)   rotate(0deg); }
  10%  { transform: translateX(5vw)   translateY(-20px)  rotate(-80deg); }
  20%  { transform: translateX(16vw)  translateY(0px)    rotate(-160deg); }
  35%  { transform: translateX(30vw)  translateY(-45px)  rotate(-300deg); }
  48%  { transform: translateX(44vw)  translateY(0px)    rotate(-420deg); }
  58%  { transform: translateX(55vw)  translateY(-30px)  rotate(-520deg); }
  70%  { transform: translateX(68vw)  translateY(0px)    rotate(-620deg); }
  82%  { transform: translateX(80vw)  translateY(-20px)  rotate(-720deg); }
  100% { transform: translateX(110vw) translateY(0px)    rotate(-880deg); }
}
@keyframes tw3 {
  0%   { transform: translateX(-10vw) translateY(0px)   rotate(0deg); }
  12%  { transform: translateX(4vw)   translateY(-55px)  rotate(100deg); }
  22%  { transform: translateX(14vw)  translateY(0px)    rotate(190deg); }
  38%  { transform: translateX(32vw)  translateY(-20px)  rotate(320deg); }
  50%  { transform: translateX(46vw)  translateY(0px)    rotate(440deg); }
  62%  { transform: translateX(58vw)  translateY(-40px)  rotate(560deg); }
  74%  { transform: translateX(72vw)  translateY(0px)    rotate(660deg); }
  88%  { transform: translateX(88vw)  translateY(-15px)  rotate(780deg); }
  100% { transform: translateX(110vw) translateY(0px)    rotate(860deg); }
}
@keyframes tw4 {
  0%   { transform: translateX(-10vw) translateY(0px)   rotate(0deg); }
  15%  { transform: translateX(8vw)   translateY(-15px)  rotate(-120deg); }
  30%  { transform: translateX(24vw)  translateY(0px)    rotate(-240deg); }
  50%  { transform: translateX(46vw)  translateY(-30px)  rotate(-400deg); }
  68%  { transform: translateX(64vw)  translateY(0px)    rotate(-540deg); }
  85%  { transform: translateX(86vw)  translateY(-10px)  rotate(-680deg); }
  100% { transform: translateX(110vw) translateY(0px)    rotate(-800deg); }
}
`;

let tumbleweedKeyframesInjected = false;
function ensureTumbleweedKeyframes() {
  if (tumbleweedKeyframesInjected) return;
  if (typeof document === "undefined") return;
  const style = document.createElement("style");
  style.textContent = TUMBLEWEED_KEYFRAMES;
  document.head.appendChild(style);
  tumbleweedKeyframesInjected = true;
}

/** Tumbleweed SVG — overlapping wobbly ellipses for a tangled bush look */
function TumbleweedSvg({ fill }: { fill: string }) {
  return (
    <svg viewBox="0 0 40 40" style={{ width: "100%", height: "100%" }}>
      <ellipse
        cx="20"
        cy="20"
        rx="16"
        ry="14"
        fill={fill}
        fillOpacity={0.15}
        stroke="#78350f"
        strokeWidth={1.2}
        opacity={0.6}
      />
      <ellipse
        cx="18"
        cy="19"
        rx="13"
        ry="15"
        fill="none"
        stroke="#92400e"
        strokeWidth={1.5}
        opacity={0.5}
      />
      <ellipse
        cx="22"
        cy="21"
        rx="14"
        ry="12"
        fill="none"
        stroke="#a16207"
        strokeWidth={1.3}
        opacity={0.5}
      />
      <ellipse
        cx="20"
        cy="18"
        rx="11"
        ry="13"
        fill="none"
        stroke="#92400e"
        strokeWidth={1}
        opacity={0.4}
      />
      <ellipse
        cx="19"
        cy="22"
        rx="12"
        ry="10"
        fill="none"
        stroke="#b45309"
        strokeWidth={1.2}
        opacity={0.4}
      />
      <circle
        cx="21"
        cy="20"
        r="8"
        fill="none"
        stroke="#78350f"
        strokeWidth={0.8}
        opacity={0.3}
      />
      <circle
        cx="19"
        cy="20"
        r="6"
        fill="none"
        stroke="#92400e"
        strokeWidth={0.8}
        opacity={0.3}
      />
      <line
        x1="10"
        y1="16"
        x2="30"
        y2="24"
        stroke="#92400e"
        strokeWidth={0.8}
        opacity={0.3}
      />
      <line
        x1="12"
        y1="26"
        x2="28"
        y2="14"
        stroke="#a16207"
        strokeWidth={0.8}
        opacity={0.25}
      />
      <line
        x1="14"
        y1="12"
        x2="26"
        y2="28"
        stroke="#78350f"
        strokeWidth={0.7}
        opacity={0.2}
      />
    </svg>
  );
}

function TumbleweedBackground() {
  ensureTumbleweedKeyframes();

  // Each weed: [sizePx, bottomPct, opacity, animName, duration, delay, fillColor]
  const weeds = [
    {
      size: 70,
      bottom: 2,
      opacity: 0.35,
      anim: "tw1",
      dur: "12s",
      delay: "0s",
      fill: "#92400e",
    },
    {
      size: 25,
      bottom: 5,
      opacity: 0.22,
      anim: "tw2",
      dur: "7s",
      delay: "3s",
      fill: "#a16207",
    },
    {
      size: 85,
      bottom: 1,
      opacity: 0.3,
      anim: "tw3",
      dur: "18s",
      delay: "8s",
      fill: "#78350f",
    },
    {
      size: 20,
      bottom: 7,
      opacity: 0.2,
      anim: "tw4",
      dur: "5s",
      delay: "1s",
      fill: "#b45309",
    },
    {
      size: 45,
      bottom: 3,
      opacity: 0.25,
      anim: "tw1",
      dur: "20s",
      delay: "14s",
      fill: "#92400e",
    },
    {
      size: 30,
      bottom: 4,
      opacity: 0.22,
      anim: "tw3",
      dur: "9s",
      delay: "18s",
      fill: "#a16207",
    },
    {
      size: 95,
      bottom: 1,
      opacity: 0.28,
      anim: "tw2",
      dur: "24s",
      delay: "22s",
      fill: "#78350f",
    },
  ];

  return (
    <div
      aria-hidden="true"
      style={{
        position: "absolute",
        inset: 0,
        pointerEvents: "none",
        zIndex: 0,
        overflow: "hidden",
      }}
    >
      {/* Tumbleweeds */}
      {weeds.map((w, i) => (
        <div
          key={i}
          style={{
            position: "absolute",
            bottom: `${w.bottom}%`,
            left: "-10vw",
            width: w.size,
            height: w.size,
            opacity: w.opacity,
            animation: `${w.anim} ${w.dur} linear ${w.delay} infinite`,
            willChange: "transform",
          }}
        >
          <TumbleweedSvg fill={w.fill} />
        </div>
      ))}
    </div>
  );
}

/** Characters used in the Matrix rain effect */
const MATRIX_ALPHABET =
  "アァカサタナハマヤャラワガザダバパイィキシチニヒミリヰギジヂビピウゥクスツヌフムユュルグズブヅプエェケセテネヘメレヱゲゼデベペオォコソトノホモヨョロヲゴゾドボポヴッン" +
  "ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
  "0123456789";

/** Canvas-based Matrix digital rain (based on react-mdr) */
function MatrixRainBackground() {
  const canvasRef = useRef<HTMLCanvasElement>(null);

  useEffect(() => {
    const canvas = canvasRef.current;
    if (!canvas) return;

    const context = canvas.getContext("2d");
    if (!context) return;

    const parent = canvas.parentElement;
    if (!parent) return;
    canvas.width = parent.clientWidth;
    canvas.height = parent.clientHeight;

    const fontSize = 16;
    const columns = Math.floor(canvas.width / fontSize);

    const maxRows = Math.ceil(canvas.height / fontSize);
    const rainDrops: number[] = [];
    for (let x = 0; x < columns; x++) {
      rainDrops[x] = Math.floor(Math.random() * maxRows);
    }

    const render = () => {
      context.fillStyle = "rgba(0, 0, 0, 0.05)";
      context.fillRect(0, 0, canvas.width, canvas.height);

      context.fillStyle = "#0F0";
      context.font = `${fontSize}px monospace`;

      for (let i = 0; i < rainDrops.length; i++) {
        const text = MATRIX_ALPHABET.charAt(
          Math.floor(Math.random() * MATRIX_ALPHABET.length),
        );
        context.fillText(text, i * fontSize, rainDrops[i] * fontSize);

        if (rainDrops[i] * fontSize > canvas.height && Math.random() > 0.975) {
          rainDrops[i] = 0;
        }
        rainDrops[i]++;
      }
    };

    const intervalId = setInterval(render, 45);

    const handleResize = () => {
      canvas.width = parent.clientWidth;
      canvas.height = parent.clientHeight;
    };
    const observer = new ResizeObserver(handleResize);
    observer.observe(parent);

    return () => {
      clearInterval(intervalId);
      observer.disconnect();
    };
  }, []);

  return (
    <div
      aria-hidden="true"
      style={{
        position: "absolute",
        inset: 0,
        pointerEvents: "none",
        zIndex: 0,
        overflow: "hidden",
      }}
    >
      <canvas
        ref={canvasRef}
        style={{ display: "block", width: "100%", height: "100%" }}
      />
    </div>
  );
}

// Initialize engine once globally
let engineInitialized = false;
let engineInitPromise: Promise<void> | null = null;

export const BackgroundAnimation = memo(function BackgroundAnimation({
  animation,
  disabled = false,
}: BackgroundAnimationProps) {
  const [init, setInit] = useState(engineInitialized);

  // Respect prefers-reduced-motion
  const prefersReducedMotion = useMemo(() => {
    if (typeof window === "undefined") return false;
    return window.matchMedia("(prefers-reduced-motion: reduce)").matches;
  }, []);

  // Initialize tsParticles engine once (pattern from official tsParticles docs)
  /* eslint-disable react-hooks/set-state-in-effect -- Official tsParticles initialization pattern */
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

  // Don't render if disabled, no animation, or user prefers reduced motion
  if (disabled || animation === "none" || prefersReducedMotion) {
    return null;
  }

  // CSS-based animations render immediately (no tsparticles dependency)
  const config = ANIMATION_CONFIGS[animation];
  if (!config && animation === "waves") {
    return <WavesBackground />;
  }
  if (!config && animation === "sun") {
    return <SunBackground />;
  }
  if (!config && animation === "tumbleweed") {
    return <TumbleweedBackground />;
  }
  if (!config && animation === "matrixRain") {
    return <MatrixRainBackground />;
  }

  // tsparticles animations need engine init
  if (!init) {
    return null;
  }

  if (!config) return null;

  // Positioned absolutely within the non-scrolling sceneArea container.
  // Messages scroll independently in a sibling layer on top.
  return (
    <div
      aria-hidden="true"
      style={{
        position: "absolute",
        inset: 0,
        pointerEvents: "none",
        zIndex: 0,
        overflow: "hidden",
      }}
    >
      <Particles
        id="game-bg-particles"
        options={config}
        style={{ position: "absolute", inset: 0 }}
      />
    </div>
  );
});
