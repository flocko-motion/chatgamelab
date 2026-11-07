/**
 * BloodDripText effect for AI messages.
 *
 * Text has a red glow that pulses like a heartbeat, with occasional
 * intensity spikes. Eerie, unsettling feel.
 * Designed for horror, zombie themes.
 */

const KEYFRAMES = `
@keyframes bloodPulse {
  0%, 100% {
    text-shadow:
      0 0 4px rgba(220, 38, 38, 0.3),
      0 2px 4px rgba(127, 29, 29, 0.2);
    filter: brightness(1);
  }
  25% {
    text-shadow:
      0 0 8px rgba(220, 38, 38, 0.5),
      0 3px 6px rgba(127, 29, 29, 0.3),
      0 0 20px rgba(185, 28, 28, 0.15);
    filter: brightness(1.05);
  }
  30% {
    text-shadow:
      0 0 12px rgba(220, 38, 38, 0.7),
      0 4px 8px rgba(127, 29, 29, 0.5),
      0 0 30px rgba(185, 28, 28, 0.25);
    filter: brightness(1.1);
  }
  35% {
    text-shadow:
      0 0 6px rgba(220, 38, 38, 0.4),
      0 2px 5px rgba(127, 29, 29, 0.3);
    filter: brightness(0.95);
  }
  50% {
    text-shadow:
      0 0 3px rgba(220, 38, 38, 0.25),
      0 2px 3px rgba(127, 29, 29, 0.15);
    filter: brightness(0.92);
  }
  75% {
    text-shadow:
      0 0 6px rgba(220, 38, 38, 0.4),
      0 3px 6px rgba(127, 29, 29, 0.25),
      0 0 16px rgba(185, 28, 28, 0.1);
    filter: brightness(1);
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

interface BloodDripTextProps {
  text: string;
}

export function BloodDripText({ text }: BloodDripTextProps) {
  ensureKeyframes();

  return (
    <span
      style={{
        animation: 'bloodPulse 4s ease-in-out infinite',
        color: 'inherit',
      }}
    >
      {text}
    </span>
  );
}
