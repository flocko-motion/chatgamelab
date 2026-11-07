import { PlayGameButton } from "@components/buttons";
import type { ObjGame } from "@/api/generated";

interface GamePlayButtonsProps {
  game: ObjGame;
  onPlay: (game: ObjGame) => void;
  label: string;
}

/**
 * Shared play button for game lists.
 * Used by AllGames, MyGames, and MyWorkshop table/card views.
 */
export function GamePlayButtons({ game, onPlay, label }: GamePlayButtonsProps) {
  return (
    <PlayGameButton
      onClick={() => onPlay(game)}
      size="xs"
      style={{ width: "100%" }}
    >
      {label}
    </PlayGameButton>
  );
}
