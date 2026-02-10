import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import {
  GuestGamePlayer,
  GuestWelcome,
  type GuestStartMode,
} from "@/features/game-player-v2";

export const Route = createFileRoute("/play/$token")({
  component: GuestPlayPage,
});

function GuestPlayPage() {
  const { token } = Route.useParams();
  const [startMode, setStartMode] = useState<GuestStartMode | null>(null);

  if (!startMode) {
    return <GuestWelcome token={token} onStart={setStartMode} />;
  }

  return (
    <GuestGamePlayer
      token={token}
      mode={startMode}
      onBack={() => setStartMode(null)}
    />
  );
}
