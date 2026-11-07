/**
 * RainbowText effect for AI messages.
 *
 * Animated rainbow gradient that sweeps across the text using
 * background-clip: text. Fully readable because the gradient
 * covers whole words, not individual characters.
 * Designed for playful, candy, circus themes.
 */

const KEYFRAMES = `
@keyframes rainbowSweep {
  0% { background-position: 0% 50%; }
  100% { background-position: 200% 50%; }
}
`;

let keyframesInjected = false;

function ensureKeyframes() {
  if (keyframesInjected) return;
  if (typeof document === "undefined") return;
  const style = document.createElement("style");
  style.textContent = KEYFRAMES;
  document.head.appendChild(style);
  keyframesInjected = true;
}

interface RainbowTextProps {
  text: string;
}

export function RainbowText({ text }: RainbowTextProps) {
  ensureKeyframes();

  return (
    <span
      style={{
        backgroundImage:
          "linear-gradient(90deg, #dc2626, #c2410c, #a16207, #15803d, #0e7490, #1d4ed8, #7c3aed, #be185d, #dc2626)",
        backgroundSize: "200% 100%",
        backgroundClip: "text",
        WebkitBackgroundClip: "text",
        color: "transparent",
        animation: "rainbowSweep 6s linear infinite",
      }}
    >
      {text}
    </span>
  );
}
