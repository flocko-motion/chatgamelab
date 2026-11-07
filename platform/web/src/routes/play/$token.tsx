import { createFileRoute } from "@tanstack/react-router";
import { PlayerPlaceholder } from "@/features/play/PlayerPlaceholder";

export const Route = createFileRoute("/play/$token")({
  component: PlayerPlaceholder,
});
