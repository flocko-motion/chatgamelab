import type { ObjGame } from '@/api/generated';

export type GameFilter = 'all' | 'own' | 'public' | 'organization' | 'favorites';

export type GameSortField = 'name' | 'createdAt' | 'modifiedAt';

export interface GameSortConfig {
  field: GameSortField;
  direction: 'asc' | 'desc';
}

export function filterGames(
  games: ObjGame[],
  filter: GameFilter,
  currentUserId?: string
): ObjGame[] {
  switch (filter) {
    case 'own':
      return games.filter((game) => game.meta?.createdBy === currentUserId);
    case 'public':
      return games.filter((game) => game.public === true);
    case 'organization':
      // TODO: Filter by organization when backend supports it
      return games;
    case 'favorites':
      // TODO: Filter by favorites when backend supports it
      return games;
    case 'all':
    default:
      return games;
  }
}

export function sortGames(games: ObjGame[], config: GameSortConfig): ObjGame[] {
  const sorted = [...games];
  const { field, direction } = config;

  sorted.sort((a, b) => {
    let aVal: string | number | undefined;
    let bVal: string | number | undefined;

    switch (field) {
      case 'name':
        aVal = a.name?.toLowerCase() ?? '';
        bVal = b.name?.toLowerCase() ?? '';
        break;
      case 'createdAt':
        aVal = a.meta?.createdAt ?? '';
        bVal = b.meta?.createdAt ?? '';
        break;
      case 'modifiedAt':
        aVal = a.meta?.modifiedAt ?? '';
        bVal = b.meta?.modifiedAt ?? '';
        break;
    }

    if (aVal === undefined && bVal === undefined) return 0;
    if (aVal === undefined) return 1;
    if (bVal === undefined) return -1;

    if (aVal < bVal) return direction === 'asc' ? -1 : 1;
    if (aVal > bVal) return direction === 'asc' ? 1 : -1;
    return 0;
  });

  return sorted;
}
