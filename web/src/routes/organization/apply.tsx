import { createFileRoute } from "@tanstack/react-router";
import { OrganizationApplyPage } from "@/features/organization-registration";

export const Route = createFileRoute("/organization/apply")({
  component: OrganizationApplyPage,
});
