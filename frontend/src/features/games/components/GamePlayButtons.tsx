import { Stack } from "@mantine/core";
import { PlayGameButton, TextButton } from "@components/buttons";
import type { Game, SessionView } from "@/api/types";

type GameRef = Pick<Game, "id" | "name">;

interface GamePlayButtonsProps {
  game: GameRef;
  hasSession: boolean;
  session: SessionView | undefined;
  onPlay: (game: GameRef) => void;
  onContinue: (session: SessionView) => void;
  onRestart: (game: GameRef, session: SessionView) => void;
  labels: {
    play: string;
    continue: string;
    restart: string;
  };
}

/**
 * Shared play/continue/restart button group for game lists.
 * Used by the game lists (GamesTable).
 */
export function GamePlayButtons({
  game,
  hasSession,
  session,
  onPlay,
  onContinue,
  onRestart,
  labels,
}: GamePlayButtonsProps) {
  if (!hasSession) {
    return (
      <PlayGameButton
        onClick={() => onPlay(game)}
        size="xs"
        style={{ width: "100%" }}
      >
        {labels.play}
      </PlayGameButton>
    );
  }

  return (
    <Stack gap={4}>
      <PlayGameButton
        onClick={() => onContinue(session!)}
        size="xs"
        style={{ width: "100%" }}
      >
        {labels.continue}
      </PlayGameButton>
      <TextButton onClick={() => onRestart(game, session!)} size="xs">
        {labels.restart}
      </TextButton>
    </Stack>
  );
}
