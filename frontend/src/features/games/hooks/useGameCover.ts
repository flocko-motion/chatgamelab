import { useState } from "react";
import { useMakeGameCover } from "@/api/hooks";
import type { GameView } from "@/api/generated/api.gen";
import { extractRawErrorCode } from "@/common/types/errorCodes";

/** At most this many characters of hint go to the server. */
export const COVER_HINT_MAX = 300;

/** The translation key telling why a cover could not be drawn. */
function coverErrorKey(error: unknown): string | null {
  if (!error) return null;
  switch (extractRawErrorCode(error)) {
    case "no_api_key":
      return "games.cover.errors.noKey";
    case "invalid_api_key":
      return "games.cover.errors.keyRejected";
    default:
      return "games.cover.errors.failed";
  }
}

/**
 * The editor's cover: the designer's hint, drawing a new cover with it, and
 * why the last attempt failed, as a translation key.
 */
export function useGameCover(gameId: string | undefined) {
  const makeCover = useMakeGameCover();
  const [hint, setHint] = useState("");

  const draw = () => {
    if (!gameId) return;
    makeCover.mutate({ id: gameId, hint: hint.trim() });
  };

  return {
    hint,
    setHint: (value: string) => setHint(value.slice(0, COVER_HINT_MAX)),
    draw,
    drawing: makeCover.isPending,
    errorKey: coverErrorKey(makeCover.error),
  };
}

/**
 * Drawing a cover when the editor saves a game that has none, without a
 * hint. It runs on after the editor closes, and stays silent when it fails:
 * the game keeps its fallback tile, and the editor's button can try again.
 */
export function useDrawCoverOnSave() {
  const makeCover = useMakeGameCover();
  return (game: GameView | undefined) => {
    if (!game?.id || game.coverAt || !game.mayEditCard) return;
    makeCover.mutate({ id: game.id });
  };
}
