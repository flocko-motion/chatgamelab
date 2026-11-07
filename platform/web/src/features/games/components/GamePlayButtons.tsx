import { Stack } from "@mantine/core";
import { PlayGameButton, TextButton } from "@components/buttons";
import type { ObjGame, DbUserSessionWithGame } from "@/api/generated";

interface GamePlayButtonsProps {
  game: ObjGame;
  hasSession: boolean;
  session: DbUserSessionWithGame | undefined;
  onPlay: (game: ObjGame) => void;
  onContinue: (session: DbUserSessionWithGame) => void;
  onRestart: (game: ObjGame, session: DbUserSessionWithGame) => void;
  labels: {
    play: string;
    continue: string;
    restart: string;
  };
}

/**
 * Shared play/continue/restart button group for game lists.
 * Used by AllGames, MyGames, and MyWorkshop table/card views.
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
