import type { Game } from "@/api/generated/api.gen";

export type GameFilter = "all" | "mine" | "workshop" | "public";

export interface WorkshopSettings {
  showPublicGames: boolean;
  showOtherParticipantsGames: boolean;
  aiQualityTier?: string;
}

export interface GamePermissions {
  canEdit: boolean;
  canDelete: boolean;
  isOwner: boolean;
}

export function getGamePermissions(
  game: Game,
  currentUserId: string | undefined,
  canEditAllWorkshopGames: boolean,
  isAdmin: boolean = false,
): GamePermissions {
  const isOwner = game.creatorId === currentUserId;
  const canEdit = isOwner || (canEditAllWorkshopGames && !!game.workshopId) || isAdmin;
  const canDelete = isOwner || (canEditAllWorkshopGames && !!game.workshopId) || isAdmin;

  return { canEdit, canDelete, isOwner };
}

export function filterGamesByWorkshopSettings(
  games: Game[],
  currentUserId: string | undefined,
  currentWorkshopId: string | undefined,
  settings: WorkshopSettings,
): Game[] {
  return games.filter((game) => {
    const isWsGame = !!game.workshopId && game.workshopId === currentWorkshopId;

    if (isWsGame) {
      // Workshop games: own always visible, others controlled by setting
      if (game.creatorId === currentUserId) return true;
      return settings.showOtherParticipantsGames;
    }

    // Non-workshop public games: controlled by showPublicGames
    if (game.public) return settings.showPublicGames;

    return false;
  });
}

export function filterGamesByUserFilter(
  games: Game[],
  filter: GameFilter,
  currentUserId: string | undefined,
  currentWorkshopId?: string,
): Game[] {
  switch (filter) {
    case "mine":
      return games.filter((game) => game.creatorId === currentUserId);
    case "workshop":
      // Filter to games in the current workshop only
      return games.filter(
        (game) => game.workshopId && game.workshopId === currentWorkshopId,
      );
    case "public":
      return games.filter((game) => game.public);
    default:
      return games;
  }
}

/**
 * Check if a game belongs to the current workshop
 */
export function isWorkshopGame(
  game: Game,
  currentWorkshopId: string | undefined,
): boolean {
  return !!game.workshopId && game.workshopId === currentWorkshopId;
}
