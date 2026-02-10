import { useCallback } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useModals } from "@mantine/modals";
import { Text } from "@mantine/core";
import { useTranslation } from "react-i18next";
import type { ObjGame, DbUserSessionWithGame } from "@/api/generated";
import { useDeleteSession } from "@/api/hooks";

/**
 * Shared hook for game play/continue/restart navigation logic.
 * Used by AllGames, MyGames, and MyWorkshop.
 */
export function useGameNavigation() {
  const navigate = useNavigate();
  const modals = useModals();
  const { t } = useTranslation("common");
  const deleteSession = useDeleteSession();

  const playGame = useCallback(
    (game: ObjGame) => {
      if (game.id) {
        navigate({ to: "/games/$gameId/play", params: { gameId: game.id } });
      }
    },
    [navigate],
  );

  const continueGame = useCallback(
    (session: DbUserSessionWithGame) => {
      if (session.id) {
        navigate({ to: `/sessions/${session.id}` as "/" });
      }
    },
    [navigate],
  );

  const restartGame = useCallback(
    (game: ObjGame, session: DbUserSessionWithGame) => {
      if (!game.id || !session.id) return;

      modals.openConfirmModal({
        title: t("myGames.restartConfirm.title"),
        children: (
          <Text size="sm">
            {t("myGames.restartConfirm.message", {
              game: game.name || t("sessions.untitledGame"),
            })}
          </Text>
        ),
        labels: {
          confirm: t("myGames.restartConfirm.confirm"),
          cancel: t("cancel"),
        },
        confirmProps: { color: "red" },
        onConfirm: async () => {
          try {
            await deleteSession.mutateAsync(session.id!);
          } catch {
            // Session may have been deleted already, ignore and continue
          }
          navigate({
            to: "/games/$gameId/play",
            params: { gameId: game.id! },
          });
        },
      });
    },
    [modals, t, deleteSession, navigate],
  );

  return { playGame, continueGame, restartGame };
}
