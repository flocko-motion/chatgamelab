import type { ObjGame } from "@/api/generated";
import type { CreateGameFormData } from "../types";

/**
 * Convert an ObjGame (from API) into CreateGameFormData for the create modal.
 * Used for the "Copy" flow - pre-populates the modal with existing game data.
 *
 * @param game - The source game to copy from
 * @param suffix - Suffix to append to the name (default: " (Copy)")
 */
export function gameToFormData(
  game: ObjGame,
  suffix = " (Copy)",
): CreateGameFormData {
  return {
    name: (game.name ?? "") + suffix,
    description: game.description ?? "",
    isPublic: false,
    systemMessageScenario: game.systemMessageScenario || undefined,
    systemMessageGameStart: game.systemMessageGameStart || undefined,
    imageStyle: game.imageStyle || undefined,
    statusFields: game.statusFields || undefined,
  };
}
