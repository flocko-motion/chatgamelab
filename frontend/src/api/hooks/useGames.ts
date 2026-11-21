import { api } from "../client/api";
import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { handleApiError } from "@/config/queryClient";
import { queryKeys } from "../queryKeys";
import { useAuth } from "@/providers/AuthProvider";
import type {
  Game,
  GameTemplate,
  GameView,
  GameViewPage,
  UpdateGameReq,
  Error as ApiError,
  CreateGameReq,
  FavouriteResp,
  GameShareResp,
  MessageResp,
  PrivateShareStatusResp,
} from "../generated/api.gen";

// Games hooks

/**
 * The filters the game list knows. Several combine (AND); none lists every
 * game the caller can see. In a workshop, that is the workshop's world.
 */
export const GAME_LIST_FILTERS = ["own", "others", "public", "favorites", "sponsored"] as const;
export type GameListFilter = (typeof GAME_LIST_FILTERS)[number];

/** The orders the game list knows. */
export const GAME_LIST_SORTS = [
  "createdAt",
  "modifiedAt",
  "name",
  "playCount",
  "visibility",
  "creator",
] as const;
export type GameListSort = (typeof GAME_LIST_SORTS)[number];

/** A game list query; every part is optional and left out when unset. */
export interface GameListParams {
  filters?: GameListFilter[];
  search?: string;
  sortBy?: GameListSort;
  sortDir?: "asc" | "desc";
  /** Page size; all matches when left out. */
  limit?: number;
  offset?: number;
}

/**
 * A page of the games the caller can see, and how many match in all. The
 * server filters, searches, sorts and pages, and decides what the caller
 * may see.
 */
export function useGames(params: GameListParams = {}) {
  const { isAuthenticated } = useAuth();
  const { filters, search, sortBy, sortDir, limit, offset } = params;

  return useQuery<GameViewPage, ApiError>({
    queryKey: queryKeys.gameList({ filters, search, sortBy, sortDir, limit, offset }),
    queryFn: () =>
      api
        .getApiGames({
          filter: filters?.length ? filters : undefined,
          search: search || undefined,
          sortBy,
          sortDir,
          limit: limit || undefined,
          offset: offset || undefined,
        })
        .then((response) => response.data),
    placeholderData: keepPreviousData,
    enabled: isAuthenticated,
  });
}

/**
 * One game as the caller sees it, with their rights on it (may* flags).
 * The server answers 404 for a game the caller may not read.
 */
export function useGame(id: string | undefined) {
  return useQuery<GameView, ApiError>({
    queryKey: [...queryKeys.games, id],
    queryFn: () => api.getApiGamesId(id!).then((response) => response.data),
    enabled: !!id,
  });
}

/**
 * What a new game starts from, with the caller's rights on it (may* flags).
 * Loaded only while `enabled`, i.e. while the create dialog is open.
 */
export function useGameTemplate(enabled: boolean) {
  return useQuery<GameTemplate, ApiError>({
    queryKey: queryKeys.gameTemplate,
    queryFn: () => api.getApiGamesNew().then((response) => response.data),
    enabled,
  });
}

/** Creates a game: name required, a field left out keeps the template's value. */
export function useCreateGame() {
  const queryClient = useQueryClient();

  return useMutation<GameView, ApiError, CreateGameReq>({
    mutationFn: (request) =>
      api.postApiGamesNew(request).then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.games });
    },
    onError: handleApiError,
  });
}

export function useUpdateGame() {
  const queryClient = useQueryClient();

  return useMutation<
    GameView,
    ApiError,
    { id: string; changes: UpdateGameReq }
  >({
    // Only the changes go out; a field left out stays as it is.
    mutationFn: ({ id, changes }) =>
      api.postApiGamesId(id, changes).then((response) => response.data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.games });
      queryClient.invalidateQueries({ queryKey: [...queryKeys.games, id] });
    },
    onError: handleApiError,
  });
}

export function useDeleteGame() {
  const queryClient = useQueryClient();

  return useMutation<GameView, ApiError, string>({
    mutationFn: (id) =>
      api.deleteApiGamesId(id).then((response) => response.data),
    onSuccess: (_, id) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.games });
      queryClient.invalidateQueries({
        queryKey: [...queryKeys.gameSessions, id],
      });
      queryClient.invalidateQueries({ queryKey: queryKeys.userSessions });
    },
    onError: handleApiError,
  });
}

export function useExportGameYaml() {
  return useMutation<string, ApiError, string>({
    mutationFn: (id) =>
      api
        .getApiGamesIdYaml(id, {
          headers: { Accept: "application/x-yaml" },
          format: "text",
        })
        .then((response) => response.data as unknown as string),
    onError: handleApiError,
  });
}


// Game Sponsoring hooks
export function useSponsorGame() {
  const queryClient = useQueryClient();

  return useMutation<
    Game,
    ApiError,
    { gameId: string; shareId: string }
  >({
    mutationFn: ({ gameId, shareId }) =>
      api.putApiGamesIdSponsor(gameId, { shareId })
        .then((response) => response.data),
    onSuccess: (_, { gameId }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.games });
      queryClient.invalidateQueries({ queryKey: [...queryKeys.games, gameId] });
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys });
    },
    onError: handleApiError,
  });
}

export function useRemoveGameSponsor() {
  const queryClient = useQueryClient();

  return useMutation<Game, ApiError, string>({
    mutationFn: (gameId) =>
      api.deleteApiGamesIdSponsor(gameId).then((response) => response.data),
    onSuccess: (_, gameId) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.games });
      queryClient.invalidateQueries({ queryKey: [...queryKeys.games, gameId] });
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys });
    },
    onError: handleApiError,
  });
}

// Favorite Games hooks: whether a game is a favorite comes with it
// (isFavorite); the favorites list is the game list's favorites filter.
export function useAddFavorite() {
  const queryClient = useQueryClient();

  return useMutation<FavouriteResp, ApiError, string>({
    mutationFn: (gameId) =>
      api.postApiGamesIdFavourite(gameId).then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.games });
    },
    onError: handleApiError,
  });
}

export function useRemoveFavorite() {
  const queryClient = useQueryClient();

  return useMutation<FavouriteResp, ApiError, string>({
    mutationFn: (gameId) =>
      api.deleteApiGamesIdFavourite(gameId).then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.games });
    },
    onError: handleApiError,
  });
}

// ── Private Share Management hooks ──────────────────────────────────────────

export function usePrivateShareStatus(gameId: string | undefined) {
  return useQuery<PrivateShareStatusResp, ApiError>({
    queryKey: [...queryKeys.games, gameId, "private-share"],
    queryFn: () =>
      api.getApiGamesIdPrivateShare(gameId!).then((response) => response.data),
    enabled: !!gameId,
  });
}

export function useCreateGameShare() {
  const queryClient = useQueryClient();

  return useMutation<
    GameShareResp,
    ApiError,
    { gameId: string; workshopId?: string; sponsorKeyShareId?: string; maxSessions?: number | null; aiQualityTier?: string | null }
  >({
    mutationFn: ({ gameId, workshopId, sponsorKeyShareId, maxSessions, aiQualityTier }) =>
      api
        .postApiGamesIdShares(gameId, {
          workshopId: workshopId ?? undefined,
          sponsorKeyShareId: sponsorKeyShareId ?? undefined,
          maxSessions: maxSessions ?? undefined,
          aiQualityTier: aiQualityTier ?? undefined,
        })
        .then((response) => response.data),
    onSuccess: (_, { gameId }) => {
      queryClient.invalidateQueries({
        queryKey: [...queryKeys.games, gameId, "private-share"],
      });
      queryClient.invalidateQueries({ queryKey: [...queryKeys.games, gameId] });
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys });
    },
    onError: handleApiError,
  });
}

export function useRevokePrivateShare() {
  const queryClient = useQueryClient();

  return useMutation<MessageResp, ApiError, { gameId: string; shareId: string }>({
    mutationFn: ({ gameId, shareId }) =>
      api
        .deleteApiGamesIdSharesShareId(gameId, shareId)
        .then((response) => response.data),
    onSuccess: (_, { gameId }) => {
      queryClient.invalidateQueries({
        queryKey: [...queryKeys.games, gameId, "private-share"],
      });
      queryClient.invalidateQueries({ queryKey: [...queryKeys.games, gameId] });
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys });
      queryClient.invalidateQueries({ queryKey: ["apiKeyGameShares"] });
    },
    onError: handleApiError,
  });
}


export function useUpdateGameShare() {
  const queryClient = useQueryClient();

  return useMutation<
    GameShareResp,
    ApiError,
    { gameId: string; shareId: string; maxSessions: number | null; aiQualityTier?: string | null }
  >({
    mutationFn: ({ gameId, shareId, maxSessions, aiQualityTier }) =>
      api
        .patchApiGamesIdSharesShareId(gameId, shareId, {
          maxSessions: maxSessions ?? undefined,
          aiQualityTier: aiQualityTier ?? undefined,
        })
        .then((response) => response.data),
    onSuccess: (_, { gameId }) => {
      queryClient.invalidateQueries({
        queryKey: [...queryKeys.games, gameId, "private-share"],
      });
      queryClient.invalidateQueries({ queryKey: [...queryKeys.games, gameId] });
      queryClient.invalidateQueries({ queryKey: ["apiKeyGameShares"] });
    },
    onError: handleApiError,
  });
}

// API Key Status hook
export function useApiKeyStatus(gameId: string | undefined) {
  return useQuery<boolean>({
    queryKey: queryKeys.apiKeyStatus(gameId!),
    queryFn: () =>
      api
        .getApiGamesIdApiKeyStatus(gameId!)
        .then((response) => response.data.available)
        .catch(() => false),
    enabled: !!gameId,
    staleTime: 0,
  });
}
