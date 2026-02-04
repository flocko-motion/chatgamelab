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
  filter?: "all" | "own" | "public" | "organization" | "favorites";
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
