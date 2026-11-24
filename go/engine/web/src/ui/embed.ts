/**
 * The playback-engine embedded on another website (#340): it shows the plain
 * game, with nothing of the platform around it and no Insights, laid out by
 * the room its frame gives it (ui/embed.css). It applies the host's CSS,
 * which lands after the page's own styles; the page's
 * Content-Security-Policy keeps what that CSS loads to the platform, the
 * stylesheet's site and a verified page's. The contract a host styles
 * against is docs/embed-styling.md.
 */
import { handedCSS, stylesheetURL } from "../embed.js";

/**
 * Whether the page is framed by another website: its parent is on another
 * origin. The platform frames it on its own origin.
 */
function embeddedElsewhere(): boolean {
  if (window.parent === window) return false;
  try {
    return window.parent.location.origin !== window.location.origin;
  } catch {
    // reading a cross-origin parent's address throws
    return true;
  }
}

/** Sets the page up for a host website, if one frames it, and applies the host's stylesheet. */
export function mountEmbed(params: URLSearchParams): void {
  const sheet = stylesheetURL(params);
  if (sheet) {
    const link = document.createElement("link");
    link.rel = "stylesheet";
    link.href = sheet;
    document.head.append(link);
  }
  if (!embeddedElsewhere()) return;
  document.body.dataset["embedded"] = "";
  listenForCSS();
  // The loader hands its CSS over once the page says it is ready.
  window.parent.postMessage({ type: "cgl-ready" }, "*");
}

/**
 * Takes CSS the host page posts, replacing what it posted before. Only the
 * parent frame is heard; who may be the parent is the page's
 * frame-ancestors.
 */
function listenForCSS(): void {
  let block: HTMLStyleElement | null = null;
  window.addEventListener("message", (event) => {
    if (event.source !== window.parent) return;
    const css = handedCSS(event.data);
    if (css === null) return;
    block ??= document.head.appendChild(document.createElement("style"));
    block.dataset["host"] = "";
    block.textContent = css;
  });
}
