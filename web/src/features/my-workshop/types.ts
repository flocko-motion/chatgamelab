import type { ObjGame, DbUserSessionWithGame } from "@/api/generated";

export type GameFilter = "all" | "mine" | "workshop" | "public";

export interface WorkshopSettings {
  showPublicGames: boolean;
  showOtherParticipantsGames: boolean;
  aiQualityTier?: string;
}

export interface GameSessionState {
  hasSession: boolean;
  session: DbUserSessionWithGame | undefined;
}

export interface GamePermissions {
  canEdit: boolean;
  canDelete: boolean;
  isOwner: boolean;
}

export function getGamePermissions(
  game: ObjGame,
  currentUserId: string | undefined,
  canEditAllWorkshopGames: boolean,
): GamePermissions {
  const isOwner = game.creatorId === currentUserId;
  const canEdit = isOwner || (canEditAllWorkshopGames && !!game.workshopId);
  const canDelete = isOwner;

  return { canEdit, canDelete, isOwner };
}

export function filterGamesByWorkshopSettings(
  games: ObjGame[],
  currentUserId: string | undefined,
  settings: WorkshopSettings,
  canEditAllWorkshopGames: boolean,
): ObjGame[] {
  if (canEditAllWorkshopGames) {
    return games;
  }

  return games.filter((game) => {
    if (game.creatorId === currentUserId) return true;
    if (game.public && !settings.showPublicGames) return false;
    if (
      !settings.showOtherParticipantsGames &&
      game.creatorId !== currentUserId
    )
      return false;
    return true;
  });
}

export function filterGamesByUserFilter(
  games: ObjGame[],
  filter: GameFilter,
  currentUserId: string | undefined,
  currentWorkshopId?: string,
): ObjGame[] {
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
  game: ObjGame,
  currentWorkshopId: string | undefined,
): boolean {
  return !!game.workshopId && game.workshopId === currentWorkshopId;
}
