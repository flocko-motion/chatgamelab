import { createFileRoute } from "@tanstack/react-router";

// A game started with its prompts adapted first; the root layout's playback
// host shows the advanced settings, then starts the session.
export const Route = createFileRoute("/games/$gameId/play/advanced")({
  component: () => null,
});
