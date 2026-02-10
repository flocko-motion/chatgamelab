/**
 * ParchmentBurnText effect for AI messages.
 *
 * Text reveals from center outward with a warm glow edge,
 * like reading a scroll by candlelight.
 * Designed for medieval, adventure, western themes.
 */

const KEYFRAMES = `
@keyframes parchmentReveal {
  0% {
    mask-size: 0% 100%;
    -webkit-mask-size: 0% 100%;
  }
  100% {
    mask-size: 200% 100%;
    -webkit-mask-size: 200% 100%;
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

interface ParchmentBurnTextProps {
  text: string;
}

export function ParchmentBurnText({ text }: ParchmentBurnTextProps) {
  ensureKeyframes();

  return (
    <span
      style={{
        textShadow: '0 0 8px rgba(255, 170, 50, 0.4), 0 0 16px rgba(255, 120, 20, 0.15)',
        maskImage: 'linear-gradient(90deg, black, black 50%, transparent 50%, transparent)',
        WebkitMaskImage: 'linear-gradient(90deg, black, black 50%, transparent 50%, transparent)',
        maskPosition: 'center',
        WebkitMaskPosition: 'center',
        maskRepeat: 'no-repeat',
        WebkitMaskRepeat: 'no-repeat',
        animation: 'parchmentReveal 1.5s ease-out forwards',
      }}
    >
      {text}
    </span>
  );
}
