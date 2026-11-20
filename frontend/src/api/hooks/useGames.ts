import { api, withQuery } from "../client/api";
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
  Error as ApiError,
  CreateGameReq,
  FavouriteResp,
  GameShareResp,
  MessageResp,
  PrivateShareStatusResp,
} from "../generated/api.gen";

// Games hooks
export interface UseGamesParams {
  search?: string;
  sortBy?:
    | "name"
    | "createdAt"
    | "modifiedAt"
    | "playCount"
    | "visibility"
    | "creator";
  sortDir?: "asc" | "desc";
  filter?:
    | "all"
    | "own"
    | "public"
    | "organization"
    | "favorites"
    | "sponsored";
}

export function useGames(params?: UseGamesParams) {
  const { isAuthenticated } = useAuth();
  const { search, sortBy, sortDir, filter } = params || {};

  return useQuery<Game[], ApiError>({
    queryKey: [...queryKeys.games, { search, sortBy, sortDir, filter }],
    queryFn: () =>
      api.getApiGames(withQuery({
            search: search || undefined,
            sortBy: sortBy || undefined,
            sortDir: sortDir || undefined,
            filter: filter || undefined,
          }))
        .then((response) => response.data),
    placeholderData: keepPreviousData,
    enabled: isAuthenticated,
  });
}

export function useGame(id: string | undefined) {

  return useQuery<Game, ApiError>({
    queryKey: [...queryKeys.games, id],
    queryFn: () => api.getApiGamesId(id!).then((response) => response.data),
    enabled: !!id,
  });
}

export function useCreateGame() {
  const queryClient = useQueryClient();

  return useMutation<Game, ApiError, CreateGameReq>({
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
    Game,
    ApiError,
    { id: string; game: Game }
  >({
    mutationFn: ({ id, game }) =>
      api.postApiGamesId(id, game).then((response) => response.data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.games });
      queryClient.invalidateQueries({ queryKey: [...queryKeys.games, id] });
    },
    onError: handleApiError,
  });
}

export function useDeleteGame() {
  const queryClient = useQueryClient();

  return useMutation<Game, ApiError, string>({
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

// Favorite Games hooks
export function useFavoriteGames() {

  return useQuery<Game[], ApiError>({
    queryKey: [...queryKeys.games, "favorites"],
    queryFn: () => api.getApiGamesFavourites().then((response) => response.data),
  });
}

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
