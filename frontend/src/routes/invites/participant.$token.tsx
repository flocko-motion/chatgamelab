import { createFileRoute, redirect } from "@tanstack/react-router";

// Re-login links handed out before the role was called Guest, printed or
// photographed as QR codes, keep working.
export const Route = createFileRoute("/invites/participant/$token")({
  beforeLoad: ({ params }) => {
    throw redirect({
      to: "/invites/guest/$token",
      params: { token: params.token },
      replace: true,
    });
  },
});
