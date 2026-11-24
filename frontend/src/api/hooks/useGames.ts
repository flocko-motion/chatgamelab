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
  GameTemplate,
  GameView,
  GameViewPage,
  UpdateGameReq,
  Error as ApiError,
  GameFilter,
  GameSort,
  CreateGameReq,
  FavouriteResp,
  ShareLinkView,
  MessageResp,
  PrivateShareStatusResp,
} from "../generated/api.gen";
import { useEntityList, type List, type ListParams, type ListPreset } from "../lists";

// Games hooks

/**
 * A filter of the game list: the server's enum, or one genre's
 * (`genreFilter`). Filters combine (AND); several genres are one filter, met
 * by a game of any of them.
 */
export type GameListFilter = GameFilter | `genre:${string}`;

/** The filter that keeps one genre's games; several are met by any of them. */
export const genreFilter = (genre: string): `genre:${string}` => `genre:${genre}`;

/** An order of the game list (the server's enum). */
export type GameListSort = GameSort;

/** A fixed game query (a dashboard card); every part is optional. */
export interface GameListParams {
  filters?: GameListFilter[];
  search?: string;
  sortBy?: GameListSort;
  sortDir?: "asc" | "desc";
  /** Page size; all matches when left out. */
  limit?: number;
  offset?: number;
}

// The list's parameters come from presets typed by the enums; the server
// refuses what it does not know.
const fetchGamePage = (params: ListParams) =>
  api.getApiGames(params as GamePageQuery).then((response) => response.data);

type GamePageQuery = Parameters<typeof api.getApiGames>[0];

/**
 * The games the caller can see, as a list: the server filters, searches,
 * sorts and pages, and decides what the caller may see. The preset's toggles
 * and the filters' `apply` are game filters.
 */
export function useGames<F extends string>(
  preset: ListPreset<F, GameListSort, GameListFilter>,
): List<GameView, F, GameListSort, GameListFilter> {
  const { isAuthenticated } = useAuth();
  return useEntityList<GameView, F, GameListSort, GameListFilter>(
    preset,
    { key: (params) => queryKeys.gameList({ ...params }), fetch: fetchGamePage },
    { enabled: isAuthenticated },
  );
}

/** One fixed page of games, e.g. a dashboard card's few. */
export function useGamePage(params: GameListParams = {}) {
  const { isAuthenticated } = useAuth();
  const { filters, search, sortBy, sortDir, limit, offset } = params;
  const request: ListParams = {
    filter: filters?.length ? filters : undefined,
    search: search || undefined,
    sortBy,
    sortDir,
    limit: limit || undefined,
    offset: offset || undefined,
  };

  return useQuery<GameViewPage, ApiError>({
    queryKey: queryKeys.gameList({ ...request }),
    queryFn: () => fetchGamePage(request),
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
 * Loads one game on demand (e.g. to copy it), through the cache `useGame`
 * shares: a fresh cached game is returned as it is.
 */
export function useFetchGame() {
  const queryClient = useQueryClient();
  return (id: string) =>
    queryClient.fetchQuery<GameView>({
      queryKey: queryKeys.game(id),
      queryFn: () => api.getApiGamesId(id).then((response) => response.data),
    });
}

/**
 * One game, read imperatively and past the cache: always what the server
 * says now. Errors are the caller's.
 */
export const fetchGame = (id: string): Promise<GameView> =>
  api.getApiGamesId(id).then((response) => response.data);

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

/**
 * Where a game's cover picture is, or null while it has none. The URL names
 * when the cover was made, so a new cover is never served from the cache.
 */
export function gameCoverUrl(game: Pick<GameView, "id" | "coverAt">): string | null {
  if (!game.coverAt) return null;
  return `/api/games/${game.id}/cover?v=${encodeURIComponent(game.coverAt)}`;
}

/**
 * Has a game's cover drawn, replacing the one it had; the hint says what it
 * should show. The server picks the key, and the call takes as long as the
 * picture does. Its errors are the caller's to show: the editor tells them
 * in place, and drawing on save stays silent.
 */
export function useMakeGameCover() {
  const queryClient = useQueryClient();

  return useMutation<GameView, ApiError, { id: string; hint?: string }>({
    mutationFn: ({ id, hint }) =>
      api.postApiGamesIdCover(id, { hint: hint ?? "" }).then((response) => response.data),
    onSuccess: (game, { id }) => {
      queryClient.setQueryData(queryKeys.game(id), game);
      queryClient.invalidateQueries({ queryKey: queryKeys.games });
    },
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
    GameView,
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

  return useMutation<GameView, ApiError, string>({
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
    ShareLinkView,
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
    ShareLinkView,
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
