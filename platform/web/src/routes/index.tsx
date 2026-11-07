import { createFileRoute, useNavigate } from "@tanstack/react-router";
import { Center, Loader } from "@mantine/core";
import { useEffect } from "react";
import { ROUTES } from "@/common/routes/routes";
import { hasExternalHomepage, getHomepageUrl } from "@/common/lib/url";
import { useAuth } from "@/providers/AuthProvider";

export const Route = createFileRoute(ROUTES.HOME)({
  component: HomeRouter,
});

/**
 * Pure routing hub — decides where the user should go:
 * - Authenticated → dashboard
 * - Not authenticated + external homepage → redirect there
 * - Not authenticated + no external homepage → /landing
 */
function HomeRouter() {
  const { isAuthenticated, isLoading } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (isLoading) return;

    if (isAuthenticated) {
      navigate({ to: ROUTES.DASHBOARD });
    } else if (hasExternalHomepage()) {
      window.location.href = getHomepageUrl();
    } else {
      navigate({ to: ROUTES.LANDING as "/" });
    }
  }, [isLoading, isAuthenticated, navigate]);

  return (
    <Center h="50vh">
      <Loader size="lg" />
    </Center>
  );
}
