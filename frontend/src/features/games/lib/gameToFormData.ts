import type { GameView } from "@/api/types";
import type { CreateGameFormData } from "../types";

/**
 * Convert an Game (from API) into CreateGameFormData for the create modal.
 * Used for the "Copy" flow - pre-populates the modal with existing game data.
 *
 * @param game - The source game to copy from
 * @param suffix - Suffix to append to the name (default: " (Copy)")
 */
export function gameToFormData(
  game: Pick<
    GameView,
    | "name"
    | "description"
    | "systemMessageScenario"
    | "systemMessageGameStart"
    | "imageStyle"
    | "voiceStyle"
    | "statusFields"
    | "genre"
    | "theme"
  >,
  suffix = " (Copy)",
): CreateGameFormData {
  return {
    name: (game.name ?? "") + suffix,
    description: game.description ?? "",
    isPublic: false,
    systemMessageScenario: game.systemMessageScenario || undefined,
    systemMessageGameStart: game.systemMessageGameStart || undefined,
    imageStyle: game.imageStyle || undefined,
    voiceStyle: game.voiceStyle || undefined,
    statusFields: game.statusFields || undefined,
    genre: game.genre || undefined,
    theme: game.theme || undefined,
  };
}
