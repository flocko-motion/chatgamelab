import type { ObjGame } from '@/api/generated';

export type SortField = 'name' | 'createdAt' | 'modifiedAt' | 'playCount' | 'visibility' | 'creator';
export type SortDirection = 'asc' | 'desc';

export interface SortConfig {
  field: SortField;
  direction: SortDirection;
}

export interface CreateGameFormData {
  name: string;
  description: string;
  isPublic: boolean;
  systemMessageScenario?: string;
  systemMessageGameStart?: string;
  imageStyle?: string;
  statusFields?: string;
}

export function sortGames(games: ObjGame[], config: SortConfig): ObjGame[] {
  return [...games].sort((a, b) => {
    let comparison = 0;
    
    switch (config.field) {
      case 'name':
        comparison = (a.name ?? '').localeCompare(b.name ?? '');
        break;
      case 'createdAt':
        comparison = new Date(a.meta?.createdAt ?? 0).getTime() - new Date(b.meta?.createdAt ?? 0).getTime();
        break;
      case 'modifiedAt':
        comparison = new Date(a.meta?.modifiedAt ?? 0).getTime() - new Date(b.meta?.modifiedAt ?? 0).getTime();
        break;
    }
    
    return config.direction === 'desc' ? -comparison : comparison;
  });
}
