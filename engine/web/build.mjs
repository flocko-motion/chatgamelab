// Bundles the player into dist/. Three entry points: the page, the audio
// worklet (which the browser loads by URL and so cannot be bundled into the
// main script), and the headless harness that plays a session from node.
import { build } from "esbuild";
import { cp, mkdir, rm } from "node:fs/promises";

await rm("dist", { recursive: true, force: true });
await mkdir("dist", { recursive: true });

await build({
  entryPoints: {
    // The player itself, drivable from node with no browser.
    play: "src/play.ts",
    // The page, and the worklet the browser loads by URL.
    main: "src/ui/main.ts",
    "mic-worklet": "src/ui/mic-worklet.ts",
  },
  outdir: "dist",
  bundle: true,
  format: "esm",
  // A dynamic import becomes its own chunk, so a heavy renderer only a few
  // themes use loads when one of them does. Chunk names carry no hash:
  // dist/ is committed, and a hash would rename the file on every rebuild.
  splitting: true,
  chunkNames: "chunks/[name]",
  target: "es2022",
  // Sourcemaps only for a debugging build. dist/ is committed so the Go module
  // builds without a JavaScript toolchain, and committing a map that changes on
  // every rebuild costs repository history for something only useful locally.
  sourcemap: process.env.DEV === "1",
  // The headless entry is read by people debugging a run, so nothing is minified
  // when DEV=1; the page is minified otherwise.
  minify: process.env.DEV !== "1",
  platform: "neutral",
  logLevel: "info",
});

// The stylesheet is bundled so each part of the page can keep its own file and
// @import it; the page still loads one.
await build({
  entryPoints: { styles: "src/ui/styles.css" },
  outdir: "dist",
  bundle: true,
  // Fonts and skin art are copied next to the page under their own names,
  // without a hash for the same reason as the chunks, so art files need names
  // unique across skins.
  loader: { ".woff2": "file", ".svg": "file" },
  assetNames: "assets/[name]",
  minify: process.env.DEV !== "1",
  logLevel: "info",
});

await cp("src/ui/index.html", "dist/index.html");
