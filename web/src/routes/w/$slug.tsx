import { createFileRoute } from "@tanstack/react-router";
import { PublicWorkshopPage } from "@/features/public-workshop/PublicWorkshopPage";

export const Route = createFileRoute("/w/$slug")({
  component: PublicWorkshopRoute,
});

function PublicWorkshopRoute() {
  const { slug } = Route.useParams();
  return <PublicWorkshopPage slug={slug} />;
}
