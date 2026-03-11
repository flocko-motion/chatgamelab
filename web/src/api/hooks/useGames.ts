import {
  keepPreviousData,
  useMutation,
  useQuery,
  useQueryClient,
} from "@tanstack/react-query";
import { handleApiError } from "@/config/queryClient";
import { useRequiredAuthenticatedApi } from "../useAuthenticatedApi";
import { queryKeys } from "../queryKeys";
import { config } from "@/config/env";
import { useAuth } from "@/providers/AuthProvider";
import type {
  ObjGame,
  HttpxErrorResponse,
  RoutesCreateGameRequest,
  RoutesGameShareResponse,
} from "../generated";

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
  const api = useRequiredAuthenticatedApi();
  const { search, sortBy, sortDir, filter } = params || {};

  return useQuery<ObjGame[], HttpxErrorResponse>({
    queryKey: [...queryKeys.games, { search, sortBy, sortDir, filter }],
    queryFn: () =>
      api.games
        .gamesList({
          search: search || undefined,
          sortBy: sortBy || undefined,
          sortDir: sortDir || undefined,
          filter: filter || undefined,
        })
        .then((response) => response.data),
    placeholderData: keepPreviousData,
  });
}

export function useGame(id: string | undefined) {
  const api = useRequiredAuthenticatedApi();

  return useQuery<ObjGame, HttpxErrorResponse>({
    queryKey: [...queryKeys.games, id],
    queryFn: () => api.games.gamesDetail(id!).then((response) => response.data),
    enabled: !!id,
  });
}

export function useCreateGame() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();

  return useMutation<ObjGame, HttpxErrorResponse, RoutesCreateGameRequest>({
    mutationFn: (request) =>
      api.games.postGames(request).then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.games });
    },
    onError: handleApiError,
  });
}

export function useUpdateGame() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();

  return useMutation<
    ObjGame,
    HttpxErrorResponse,
    { id: string; game: ObjGame }
  >({
    mutationFn: ({ id, game }) =>
      api.games.gamesCreate(id, game).then((response) => response.data),
    onSuccess: (_, { id }) => {
      queryClient.invalidateQueries({ queryKey: queryKeys.games });
      queryClient.invalidateQueries({ queryKey: [...queryKeys.games, id] });
    },
    onError: handleApiError,
  });
}

export function useDeleteGame() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();

  return useMutation<ObjGame, HttpxErrorResponse, string>({
    mutationFn: (id) =>
      api.games.gamesDelete(id).then((response) => response.data),
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
  const { getAccessToken } = useAuth();

  return useMutation<string, HttpxErrorResponse, string>({
    mutationFn: async (id) => {
      const token = await getAccessToken();
      const response = await fetch(`${config.API_BASE_URL}/games/${id}/yaml`, {
        method: "GET",
        headers: {
          Authorization: `Bearer ${token}`,
          Accept: "application/x-yaml",
        },
      });
      if (!response.ok) {
        const error = await response.json();
        throw error;
      }
      return response.text();
    },
    onError: handleApiError,
  });
}


// Game Sponsoring hooks
export function useSponsorGame() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();

  return useMutation<
    ObjGame,
    HttpxErrorResponse,
    { gameId: string; shareId: string }
  >({
    mutationFn: ({ gameId, shareId }) =>
      api.games
        .sponsorUpdate(gameId, { shareId })
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
  const api = useRequiredAuthenticatedApi();

  return useMutation<ObjGame, HttpxErrorResponse, string>({
    mutationFn: (gameId) =>
      api.games.sponsorDelete(gameId).then((response) => response.data),
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
  const api = useRequiredAuthenticatedApi();

  return useQuery<ObjGame[], HttpxErrorResponse>({
    queryKey: [...queryKeys.games, "favorites"],
    queryFn: () => api.games.favouritesList().then((response) => response.data),
  });
}

export function useAddFavorite() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();

  return useMutation<Record<string, boolean>, HttpxErrorResponse, string>({
    mutationFn: (gameId) =>
      api.games.favouriteCreate(gameId).then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.games });
    },
    onError: handleApiError,
  });
}

export function useRemoveFavorite() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();

  return useMutation<Record<string, boolean>, HttpxErrorResponse, string>({
    mutationFn: (gameId) =>
      api.games.favouriteDelete(gameId).then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.games });
    },
    onError: handleApiError,
  });
}

// ── Private Share Management hooks ──────────────────────────────────────────

export interface EnrichedGameShare {
  id: string;
  gameId: string;
  token: string;
  apiKeyShareId: string;
  institutionId: string | null;
  workshopId: string | null;
  remaining: number | null;
  createdBy: string | null;
  createdAt: string;
  shareUrl: string;
  source: "workshop" | "organization" | "personal";
  workshopName?: string;
}

export interface PrivateShareStatus {
  shares: EnrichedGameShare[];
}

export function usePrivateShareStatus(gameId: string | undefined) {
  const { getAccessToken } = useAuth();

  return useQuery<PrivateShareStatus>({
    queryKey: [...queryKeys.games, gameId, "private-share"],
    queryFn: async () => {
      const token = await getAccessToken();
      const response = await fetch(
        `${config.API_BASE_URL}/games/${gameId}/private-share`,
        {
          headers: { Authorization: `Bearer ${token}` },
          credentials: "include",
        },
      );
      if (!response.ok) throw new Error("Failed to fetch private share status");
      return response.json();
    },
    enabled: !!gameId,
  });
}

export function useCreateGameShare() {
  const queryClient = useQueryClient();
  const { getAccessToken } = useAuth();

  return useMutation<
    RoutesGameShareResponse,
    Error,
    { gameId: string; workshopId?: string; sponsorKeyShareId?: string; maxSessions?: number | null }
  >({
    mutationFn: async ({ gameId, workshopId, sponsorKeyShareId, maxSessions }) => {
      const token = await getAccessToken();
      const response = await fetch(
        `${config.API_BASE_URL}/games/${gameId}/shares`,
        {
          method: "POST",
          headers: {
            Authorization: `Bearer ${token}`,
            "Content-Type": "application/json",
          },
          credentials: "include",
          body: JSON.stringify({
            workshopId: workshopId ?? null,
            sponsorKeyShareId: sponsorKeyShareId ?? null,
            maxSessions: maxSessions ?? null,
          }),
        },
      );
      if (!response.ok) {
        const err = await response.json().catch(() => ({}));
        throw new Error(err.message || "Failed to create share link");
      }
      return response.json();
    },
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
  const { getAccessToken } = useAuth();

  return useMutation<void, Error, { gameId: string; shareId: string }>({
    mutationFn: async ({ gameId, shareId }) => {
      const token = await getAccessToken();
      const response = await fetch(
        `${config.API_BASE_URL}/games/${gameId}/shares/${shareId}`,
        {
          method: "DELETE",
          headers: { Authorization: `Bearer ${token}` },
          credentials: "include",
        },
      );
      if (!response.ok) {
        const err = await response.json().catch(() => ({}));
        throw new Error(err.message || "Failed to revoke private share");
      }
    },
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
  const { getAccessToken } = useAuth();

  return useMutation<
    void,
    Error,
    { gameId: string; shareId: string; maxSessions: number | null }
  >({
    mutationFn: async ({ gameId, shareId, maxSessions }) => {
      const token = await getAccessToken();
      const response = await fetch(
        `${config.API_BASE_URL}/games/${gameId}/shares/${shareId}`,
        {
          method: "PATCH",
          headers: {
            Authorization: `Bearer ${token}`,
            "Content-Type": "application/json",
          },
          credentials: "include",
          body: JSON.stringify({ maxSessions }),
        },
      );
      if (!response.ok) {
        const err = await response.json().catch(() => ({}));
        throw new Error(err.message || "Failed to update share");
      }
    },
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
  const { getAccessToken } = useAuth();

  return useQuery<boolean>({
    queryKey: queryKeys.apiKeyStatus(gameId!),
    queryFn: async () => {
      const token = await getAccessToken();
      const response = await fetch(
        `${config.API_BASE_URL}/games/${gameId}/api-key-status`,
        {
          headers: {
            Authorization: `Bearer ${token}`,
          },
          credentials: "include",
        },
      );
      if (!response.ok) return false;
      const data: { available: boolean } = await response.json();
      return data.available;
    },
    enabled: !!gameId,
    staleTime: 0,
  });
}
