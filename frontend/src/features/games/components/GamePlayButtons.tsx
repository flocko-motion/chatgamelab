import { Stack } from "@mantine/core";
import { PlayGameButton, TextButton } from "@components/buttons";
import type { GameView, SessionView } from "@/api/types";
import { blocksContinue, unplayable } from "@/common/lib/playability";
import { UnplayableNote } from "@components/UnplayableNote";

type GameRef = Pick<GameView, "id" | "name" | "playability">;

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
 * Used by the game lists (GamesTable) and the game page. A game the caller
 * cannot start has its Play and Restart off, with the reason beside them.
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
  const problem = unplayable(game);
  // Continuing the caller's own session is held only by what holds a
  // resume too; starting anew by every problem.
  if (problem && (!hasSession || blocksContinue(problem))) {
    return (
      <Stack gap={4}>
        <PlayGameButton disabled size="xs" style={{ width: "100%" }}>
          {hasSession ? labels.continue : labels.play}
        </PlayGameButton>
        <UnplayableNote problem={problem} />
      </Stack>
    );
  }

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
      {problem ? (
        <UnplayableNote problem={problem} />
      ) : (
        <TextButton onClick={() => onRestart(game, session!)} size="xs">
          {labels.restart}
        </TextButton>
      )}
    </Stack>
  );
}
