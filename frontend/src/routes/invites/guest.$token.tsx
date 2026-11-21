import { createFileRoute } from "@tanstack/react-router";
import { GuestLinkPage } from "@/features/auth";

export const Route = createFileRoute("/invites/guest/$token")({
  component: GuestLoginRoute,
});

function GuestLoginRoute() {
  const { token } = Route.useParams();
  return <GuestLinkPage key={token} token={token} />;
}
