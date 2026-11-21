import { Stack } from "@mantine/core";
import { GamesTable } from "@/features/games";
import { useAuth } from "@/providers/AuthProvider";
import { hasRole, Role } from "@/common/lib/roles";
import { useWorkshopEvents, useGamesCacheUpdater, useWorkshop } from "@/api/hooks";
import { WorkshopHeader } from "./WorkshopHeader";
import { WorkshopPausedOverlay } from "./WorkshopPausedOverlay";

/** The workshop the caller is in: its header and its games. */
export function MyWorkshop() {
  const { backendUser, retryBackendFetch } = useAuth();
  const { refetchLists, addGameToCache, updateGameInCache, removeGameFromCache } =
    useGamesCacheUpdater();

  const workshop = backendUser?.role?.workshop;
  // The workshop's settings come from reading the workshop itself.
  const { data: settings, refetch: refetchSettings } = useWorkshop(workshop?.id);
  const moderates = hasRole(backendUser, Role.Head) || hasRole(backendUser, Role.Staff);
  // A paused workshop stops Guests and individuals, not its staff.
  const isPausedForUser = !moderates && (settings?.isPaused ?? false);

  // Settings changes and games created, changed or deleted by others. The
  // settings decide what the server lists, so they refetch the list too.
  useWorkshopEvents({
    workshopId: workshop?.id,
    onSettingsUpdate: () => {
      retryBackendFetch();
      void refetchSettings();
      refetchLists();
    },
    onGameCreated: addGameToCache,
    onGameUpdated: updateGameInCache,
    onGameDeleted: removeGameFromCache,
  });

  return (
    // at least the content height, so the paused overlay covers the page
    <Stack gap="lg" mih="var(--app-content-height)" style={{ position: "relative" }}>
      {isPausedForUser && <WorkshopPausedOverlay />}
      <WorkshopHeader
        workshopName={workshop?.name}
        organizationName={backendUser?.role?.institution?.name}
        workshopId={workshop?.id}
        showMembers={moderates}
      />
      <GamesTable preset="workshop" />
    </Stack>
  );
}
