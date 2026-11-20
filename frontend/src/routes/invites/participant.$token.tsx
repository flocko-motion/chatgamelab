import { createFileRoute } from "@tanstack/react-router";
import { ParticipantLinkPage } from "@/features/auth";

export const Route = createFileRoute("/invites/participant/$token")({
  component: ParticipantLoginRoute,
});

function ParticipantLoginRoute() {
  const { token } = Route.useParams();
  return <ParticipantLinkPage key={token} token={token} />;
}
