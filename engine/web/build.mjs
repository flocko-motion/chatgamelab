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

await cp("src/ui/index.html", "dist/index.html");
await cp("src/ui/styles.css", "dist/styles.css");
