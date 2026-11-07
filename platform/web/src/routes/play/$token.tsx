import { createFileRoute } from "@tanstack/react-router";
import { useState } from "react";
import {
  GuestGamePlayer,
  GuestWelcome,
  type GuestCopyInfo,
  type GuestStartMode,
} from "@/features/game-player-v2";

export const Route = createFileRoute("/play/$token")({
  component: GuestPlayPage,
});

function GuestPlayPage() {
  const { token } = Route.useParams();
  const [startMode, setStartMode] = useState<GuestStartMode | null>(null);
  const [copyInfo, setCopyInfo] = useState<GuestCopyInfo | null>(null);

  if (!startMode) {
    return (
      <GuestWelcome
        token={token}
        onStart={setStartMode}
        onInfoLoaded={setCopyInfo}
      />
    );
  }

  return (
    <GuestGamePlayer
      token={token}
      mode={startMode}
      copyInfo={copyInfo}
      onBack={() => setStartMode(null)}
    />
  );
}
