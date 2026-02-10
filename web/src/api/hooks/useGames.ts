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

export function useCloneGame() {
  const queryClient = useQueryClient();
  const api = useRequiredAuthenticatedApi();

  return useMutation<ObjGame, HttpxErrorResponse, string>({
    mutationFn: (id) =>
      api.games.cloneCreate(id).then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.games });
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

export function useImportGameYaml() {
  const queryClient = useQueryClient();
  const { getAccessToken } = useAuth();

  return useMutation<ObjGame, HttpxErrorResponse, { id: string; yaml: string }>(
    {
      mutationFn: async ({ id, yaml }) => {
        const token = await getAccessToken();
        const headers: Record<string, string> = {
          "Content-Type": "application/x-yaml",
        };
        // Only add Authorization header if we have a token (participants use cookies)
        if (token) {
          headers["Authorization"] = `Bearer ${token}`;
        }
        const response = await fetch(
          `${config.API_BASE_URL}/games/${id}/yaml`,
          {
            method: "PUT",
            headers,
            credentials: "include", // Include cookies for participant auth
            body: yaml,
          },
        );

        if (!response.ok) {
          const error = await response
            .json()
            .catch(() => ({ message: "Import failed" }));
          throw { ...error, status: response.status };
        }

        return response.json();
      },
      onSuccess: (_, { id }) => {
        queryClient.invalidateQueries({ queryKey: queryKeys.games });
        queryClient.invalidateQueries({ queryKey: [...queryKeys.games, id] });
      },
      onError: handleApiError,
    },
  );
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

export interface PrivateShareStatus {
  enabled: boolean;
  shareUrl?: string;
  token?: string;
  remaining: number | null;
  privateSponsoredApiKeyShareId?: string;
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

export function useEnablePrivateShare() {
  const queryClient = useQueryClient();
  const { getAccessToken } = useAuth();

  return useMutation<
    PrivateShareStatus,
    Error,
    { gameId: string; sponsorKeyShareId: string; maxSessions?: number | null }
  >({
    mutationFn: async ({ gameId, sponsorKeyShareId, maxSessions }) => {
      const token = await getAccessToken();
      const response = await fetch(
        `${config.API_BASE_URL}/games/${gameId}/private-share`,
        {
          method: "POST",
          headers: {
            Authorization: `Bearer ${token}`,
            "Content-Type": "application/json",
          },
          credentials: "include",
          body: JSON.stringify({
            sponsorKeyShareId,
            maxSessions: maxSessions ?? null,
          }),
        },
      );
      if (!response.ok) {
        const err = await response.json().catch(() => ({}));
        throw new Error(err.message || "Failed to enable private share");
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

  return useMutation<PrivateShareStatus, Error, string>({
    mutationFn: async (gameId) => {
      const token = await getAccessToken();
      const response = await fetch(
        `${config.API_BASE_URL}/games/${gameId}/private-share`,
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
      return response.json();
    },
    onSuccess: (_, gameId) => {
      queryClient.invalidateQueries({
        queryKey: [...queryKeys.games, gameId, "private-share"],
      });
      queryClient.invalidateQueries({ queryKey: [...queryKeys.games, gameId] });
      queryClient.invalidateQueries({ queryKey: queryKeys.apiKeys });
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
