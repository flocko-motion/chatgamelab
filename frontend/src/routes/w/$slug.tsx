import { createFileRoute } from "@tanstack/react-router";
import { PublicWorkshopPage } from "@/features/public-workshop/PublicWorkshopPage";

interface PublicWorkshopSearch {
  /** Game to copy, set by the "Kopieren" link that opens this page in a new tab. */
  copy?: string;
}

export const Route = createFileRoute("/w/$slug")({
  component: PublicWorkshopRoute,
  validateSearch: (search: Record<string, unknown>): PublicWorkshopSearch => ({
    copy: typeof search.copy === "string" ? search.copy : undefined,
  }),
});

function PublicWorkshopRoute() {
  const { slug } = Route.useParams();
  const { copy } = Route.useSearch();
  return <PublicWorkshopPage slug={slug} copyGameId={copy} />;
}
