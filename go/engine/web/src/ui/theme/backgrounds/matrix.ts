/** The Matrix's digital rain on a canvas, after react-mdr. */
import { backdrop } from "./dom.js";

const alphabet =
  "アァカサタナハマヤャラワガザダバパイィキシチニヒミリヰギジヂビピウゥクスツヌフムユュルグズブヅプエェケセテネヘメレヱゲゼデベペオォコソトノホモヨョロヲゴゾドボポヴッン" +
  "ABCDEFGHIJKLMNOPQRSTUVWXYZ" +
  "0123456789";

const fontSize = 16;
const frameMs = 45;

export function mountMatrixRain(layer: HTMLElement): () => void {
  const root = backdrop();
  const canvas = document.createElement("canvas");
  canvas.className = "bg-matrix";
  root.append(canvas);
  layer.append(root);

  const context = canvas.getContext("2d");
  if (!context) return () => {};

  canvas.width = root.clientWidth;
  canvas.height = root.clientHeight;

  // Columns are counted once, at mount: a resize clears the canvas but keeps
  // the drops where they were.
  const columns = Math.floor(canvas.width / fontSize);
  const maxRows = Math.ceil(canvas.height / fontSize);
  const drops = Array.from({ length: columns }, () => Math.floor(Math.random() * maxRows));

  const render = (): void => {
    // A translucent wash each frame is what leaves the fading trails.
    context.fillStyle = "rgba(0, 0, 0, 0.05)";
    context.fillRect(0, 0, canvas.width, canvas.height);

    context.fillStyle = "#0F0";
    context.font = `${fontSize}px monospace`;

    drops.forEach((row, column) => {
      const glyph = alphabet.charAt(Math.floor(Math.random() * alphabet.length));
      context.fillText(glyph, column * fontSize, row * fontSize);
      drops[column] = row * fontSize > canvas.height && Math.random() > 0.975 ? 1 : row + 1;
    });
  };

  const interval = setInterval(render, frameMs);
  const observer = new ResizeObserver(() => {
    canvas.width = root.clientWidth;
    canvas.height = root.clientHeight;
  });
  observer.observe(root);

  return () => {
    clearInterval(interval);
    observer.disconnect();
  };
}
