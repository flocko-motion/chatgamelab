/**
 * The animated layer behind the scene, ported from the platform's player
 * (game-player-v2/components/BackgroundAnimation.tsx) so a theme moves the
 * same in both. Styles are in backgrounds.css.
 */
import type { BackgroundAnimation } from "../types.js";
import { backdrop, div } from "./dom.js";
import { mountMatrixRain } from "./matrix.js";
import type { ParticleAnimation } from "./particles.js";
import { mountSun } from "./sun.js";
import { mountTumbleweed } from "./tumbleweed.js";
import { mountWaves } from "./waves.js";

/** Starts `animation` inside `layer` and returns what stops it and clears the layer. */
export function mountBackground(layer: HTMLElement, animation: BackgroundAnimation): () => void {
  if (animation === "none" || matchMedia("(prefers-reduced-motion: reduce)").matches) return () => {};

  const stop = start(layer, animation);
  return () => {
    stop();
    layer.replaceChildren();
  };
}

function start(layer: HTMLElement, animation: Exclude<BackgroundAnimation, "none">): () => void {
  switch (animation) {
    case "waves":
      mountWaves(layer);
      return () => {};
    case "sun":
      mountSun(layer);
      return () => {};
    case "tumbleweed":
      mountTumbleweed(layer);
      return () => {};
    case "matrixRain":
      return mountMatrixRain(layer);
    default:
      return mountParticles(layer, animation);
  }
}

function mountParticles(layer: HTMLElement, animation: ParticleAnimation): () => void {
  const root = backdrop();
  const host = div("bg-particles");
  root.append(host);
  layer.append(root);

  // The disposer can run while tsParticles is still downloading or starting,
  // so whichever finishes second tears the container down.
  let disposed = false;
  let stop = (): void => {};
  import("./particles.js")
    .then(({ startParticles }) => startParticles(host, animation, () => disposed))
    .then((container) => {
      if (!container) return;
      if (disposed) container.destroy();
      else stop = () => container.destroy();
    })
    .catch((error: unknown) => console.warn("background animation failed to start", error));

  return () => {
    disposed = true;
    stop();
  };
}
