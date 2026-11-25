/**
 * The loader a host page includes to embed a game (#340), served as
 * embed.js beside the playback-engine:
 *
 *   <style data-cgl> [data-part="input"] { border-radius: 0 } </style>
 *   <div data-cgl-embed data-token="…" style="height: 32rem"></div>
 *   <script src="https://…/api/engine/playback-engine/embed.js" async></script>
 *
 * It turns each [data-cgl-embed] into the playback-engine's iframe, filling
 * the element, and hands every <style data-cgl> block of the page over to
 * it. The game lays itself out by the room it gets. It stays in its frame:
 * the host page never reads the conversation. docs/embedding.md is the
 * contract.
 */

const script = document.currentScript as HTMLScriptElement | null;
const base = new URL(".", script?.src ?? location.href);
const origin = base.origin;

/** The host page's CSS for its embeds: every <style data-cgl> block. */
function hostCSS(): string {
  return Array.from(document.querySelectorAll<HTMLStyleElement>("style[data-cgl]"))
    .map((style) => style.textContent ?? "")
    .join("\n");
}

/** The playback-engine's address for one embed, as its element asks. */
function address(host: HTMLElement): string {
  const url = new URL(base.href);
  url.search = "";
  url.searchParams.set("new", "");
  url.searchParams.set("share", host.dataset["token"] ?? "");
  for (const name of ["player", "request", "lang", "css", "mode"]) {
    const value = host.dataset[name];
    if (value) url.searchParams.set(name, value);
  }
  return url.toString();
}

function embed(host: HTMLElement): void {
  if (host.dataset["cglMounted"] !== undefined || !host.dataset["token"]) return;
  host.dataset["cglMounted"] = "";
  const frame = document.createElement("iframe");
  frame.src = address(host);
  frame.title = host.dataset["title"] ?? "Game";
  frame.allow = "microphone; autoplay";
  frame.style.border = "0";
  frame.style.width = "100%";
  frame.style.display = "block";
  // An element the host gave a height is filled; otherwise the frame takes one of its own.
  frame.style.height = host.offsetHeight > 0 ? "100%" : "36rem";
  host.append(frame);

  window.addEventListener("message", (event) => {
    if (event.source !== frame.contentWindow || event.origin !== origin) return;
    if ((event.data as { type?: unknown })?.type === "cgl-ready") {
      frame.contentWindow?.postMessage({ type: "cgl-css", css: hostCSS() }, origin);
    }
  });
}

function mountAll(): void {
  document.querySelectorAll<HTMLElement>("[data-cgl-embed]").forEach(embed);
}

if (document.readyState === "loading") document.addEventListener("DOMContentLoaded", mountAll);
else mountAll();

// A module of its own, so its names stay out of the page's globals.
export {};
