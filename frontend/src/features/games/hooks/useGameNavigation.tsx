import { useCallback } from "react";
import { useNavigate } from "@tanstack/react-router";
import { useModals } from "@mantine/modals";
import { Group, Stack, Text } from "@mantine/core";
import { IconAdjustmentsHorizontal } from "@tabler/icons-react";
import { ActionButton, TextButton } from "@components/buttons";
import { BodyText } from "@components/typography";
import { useTranslation } from "react-i18next";
import type { Game, SessionView } from "@/api/types";

/**
 * What playing needs of a game: a full game or a list card both do. A
 * game's view says whether the caller may adapt its prompts before it
 * starts (admins, heads and staff).
 */
type GameRef = Pick<Game, "id" | "name"> & { mayPresetPrompts?: boolean };

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

  /** Starts a game with its prompts adapted first, in the advanced settings. */
  const playGameAdvanced = useCallback(
    (game: GameRef) => {
      if (game.id) {
        navigate({ to: "/games/$gameId/play/advanced", params: { gameId: game.id } });
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
   * Opens a game the way a click on its tile or its Play asks: a game the
   * caller has a session of asks first whether to continue it or start
   * anew; one whose prompts the caller may adapt also offers the advanced
   * settings, and asks even without a session. Any other starts. A game
   * that cannot be started anew, but can be continued, continues without
   * asking.
   */
  const openGame = useCallback(
    (game: GameRef, session: SessionView | undefined, canStartNew: boolean) => {
      const resumable = session?.id ? session : undefined;
      if (resumable && !canStartNew) {
        continueGame(resumable);
        return;
      }
      const advanced = !!game.mayPresetPrompts;
      if (!resumable && !advanced) {
        playGame(game);
        return;
      }
      const name = game.name || t("sessions.untitledGame");
      const choose = (go: () => void) => () => {
        modals.closeModal(id);
        go();
      };
      const id = modals.openModal({
        title: t(resumable ? "allGames.resumeChoice.title" : "allGames.startChoice.title"),
        children: (
          <Stack gap="md">
            <BodyText size="sm">
              {resumable
                ? t("allGames.resumeChoice.message", { game: name })
                : t("allGames.startChoice.message", { game: name })}
            </BodyText>
            {/* The advanced settings start anew too, with adapted prompts:
                set apart on the left, the usual choices on the right. */}
            <Group justify={advanced ? "space-between" : "flex-end"} gap="sm" wrap="wrap">
              {advanced && (
                <TextButton
                  size="xs"
                  leftSection={<IconAdjustmentsHorizontal size={16} />}
                  onClick={choose(() => playGameAdvanced(game))}
                >
                  {t("allGames.resumeChoice.advanced")}
                </TextButton>
              )}
              <Group gap="sm" ml="auto">
                {/* Starting anew replaces the player's earlier sessions of the game. */}
                {resumable ? (
                  <TextButton onClick={choose(() => playGame(game))}>
                    {t("allGames.resumeChoice.startNew")}
                  </TextButton>
                ) : (
                  <ActionButton onClick={choose(() => playGame(game))}>
                    {t("allGames.startChoice.start")}
                  </ActionButton>
                )}
                {resumable && (
                  <ActionButton onClick={choose(() => continueGame(resumable))}>
                    {t("allGames.resumeChoice.continue")}
                  </ActionButton>
                )}
              </Group>
            </Group>
          </Stack>
        ),
      });
    },
    [modals, t, playGame, playGameAdvanced, continueGame],
  );

  return { playGame, playGameAdvanced, continueGame, restartGame, openGame };
}
