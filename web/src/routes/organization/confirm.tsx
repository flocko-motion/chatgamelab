import { createFileRoute } from "@tanstack/react-router";
import { OrganizationConfirmPage } from "@/features/organization-registration";

export const Route = createFileRoute("/organization/confirm")({
  validateSearch: (search: Record<string, unknown>) => ({
    token: typeof search.token === "string" ? search.token : "",
  }),
  component: ConfirmRoute,
});

function ConfirmRoute() {
  const { token } = Route.useSearch();
  return <OrganizationConfirmPage token={token} />;
}
