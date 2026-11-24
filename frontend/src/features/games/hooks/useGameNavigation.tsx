import { useCallback } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useModals } from "@mantine/modals";
import { Group, Stack, Text } from "@mantine/core";
import { ActionButton, TextButton } from "@components/buttons";
import { BodyText } from "@components/typography";
import { useTranslation } from "react-i18next";
import type { Game, SessionView } from "@/api/types";

/** What playing needs of a game: a full game or a list card both do. */
type GameRef = Pick<Game, "id" | "name">;

/**
 * Shared hook for game play/continue/restart navigation logic.
 * Used by the game lists (GamesTable) and the game page.
 */
export function useGameNavigation() {
  const navigate = useNavigate();
  const modals = useModals();
  const { t } = useTranslation("common");

  const playGame = useCallback(
    (game: GameRef) => {
      if (game.id) {
        navigate({ to: "/games/$gameId/play", params: { gameId: game.id } });
      }
    },
    [navigate],
  );

  const continueGame = useCallback(
    (session: SessionView) => {
      if (session.id) {
        navigate({ to: `/sessions/${session.id}` as "/" });
      }
    },
    [navigate],
  );

  const restartGame = useCallback(
    (game: GameRef, session: SessionView) => {
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
        // Starting a game is starting it anew: the new session replaces the
        // player's earlier ones of the game.
        onConfirm: () => {
          navigate({
            to: "/games/$gameId/play",
            params: { gameId: game.id! },
          });
        },
      });
    },
    [modals, t, navigate],
  );

  /**
   * Opens a game the way a click on its tile asks: a game the caller has a
   * session of asks first whether to continue it or start anew; any other
   * starts. A game that cannot be started anew, but can be continued,
   * continues without asking.
   */
  const openGame = useCallback(
    (game: GameRef, session: SessionView | undefined, canStartNew: boolean) => {
      if (!session?.id) {
        playGame(game);
        return;
      }
      if (!canStartNew) {
        continueGame(session);
        return;
      }
      const id = modals.openModal({
        title: t("allGames.resumeChoice.title"),
        children: (
          <Stack gap="md">
            <BodyText size="sm">
              {t("allGames.resumeChoice.message", {
                game: game.name || t("sessions.untitledGame"),
              })}
            </BodyText>
            <Group justify="flex-end" gap="sm">
              <TextButton
                onClick={() => {
                  modals.closeModal(id);
                  // Starting anew replaces the player's earlier sessions of the game.
                  playGame(game);
                }}
              >
                {t("allGames.resumeChoice.startNew")}
              </TextButton>
              <ActionButton
                onClick={() => {
                  modals.closeModal(id);
                  continueGame(session);
                }}
              >
                {t("allGames.resumeChoice.continue")}
              </ActionButton>
            </Group>
          </Stack>
        ),
      });
    },
    [modals, t, playGame, continueGame],
  );

  return { playGame, continueGame, restartGame, openGame };
}
