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
 * - External homepage configured → redirect there
 * - Authenticated → dashboard
 * - Not authenticated → /landing (built-in landing page)
 */
function HomeRouter() {
  const { isAuthenticated, isLoading } = useAuth();
  const navigate = useNavigate();

  useEffect(() => {
    if (isLoading) return;

    if (hasExternalHomepage()) {
      // External homepage: always redirect there (logged-in users go to dashboard via the external site)
      window.location.href = getHomepageUrl();
    } else if (isAuthenticated) {
      navigate({ to: ROUTES.DASHBOARD });
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
