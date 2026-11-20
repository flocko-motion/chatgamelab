import { createFileRoute } from "@tanstack/react-router";

// The root layout renders the playback-engine's host for every play route, so
// the frame stays mounted when the address changes to the session's.
export const Route = createFileRoute("/sessions/$sessionId")({
  component: () => null,
});
