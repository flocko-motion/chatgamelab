import { useCallback } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useModals } from "@mantine/modals";
import { Text } from "@mantine/core";
import { useTranslation } from "react-i18next";
import type { Game, UserSessionWithGame } from "@/api/generated/api.gen";

/** What playing needs of a game: a full game or a list card both do. */
type GameRef = Pick<Game, "id" | "name">;
import { useDeleteSession } from "@/api/hooks";
import { forgetRememberedSession } from "@/features/play/playback";

/**
 * Shared hook for game play/continue/restart navigation logic.
 * Used by the game lists (GamesTable).
 */
export function useGameNavigation() {
  const navigate = useNavigate();
  const modals = useModals();
  const { t } = useTranslation("common");
  const deleteSession = useDeleteSession();

  const playGame = useCallback(
    (game: GameRef) => {
      if (game.id) {
        navigate({ to: "/games/$gameId/play", params: { gameId: game.id } });
      }
    },
    [navigate],
  );

  const continueGame = useCallback(
    (session: UserSessionWithGame) => {
      if (session.id) {
        navigate({ to: `/sessions/${session.id}` as "/" });
      }
    },
    [navigate],
  );

  const restartGame = useCallback(
    (game: GameRef, session: UserSessionWithGame) => {
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
          forgetRememberedSession(game.id!);
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
