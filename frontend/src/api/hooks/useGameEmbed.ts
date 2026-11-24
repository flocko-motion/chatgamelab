import { useMutation, useQuery, useQueryClient } from "@tanstack/react-query";
import { handleApiError } from "@/config/queryClient";
import { api } from "../client/api";
import { queryKeys } from "../queryKeys";
import type { Error as ApiError, GameEmbedView, SetGameEmbedReq } from "../generated/api.gen";

/**
 * A game's embed link: its token, whether it is on, who may frame it and
 * what pays. Null while the game has none. The server answers it to whoever
 * may share the game; anyone else gets an error, and no link.
 */
export function useGameEmbed(gameId: string | undefined, enabled = true) {
  return useQuery<GameEmbedView | null, ApiError>({
    queryKey: queryKeys.gameEmbed(gameId ?? ""),
    queryFn: () => api.getApiGamesIdEmbed(gameId!).then((r) => r.data.embed ?? null),
    enabled: !!gameId && enabled,
    retry: false,
  });
}

/** Sets a game's embed link, making it the first time; its token never changes. */
export function useSetGameEmbed(gameId: string) {
  const queryClient = useQueryClient();
  return useMutation<GameEmbedView | null, ApiError, SetGameEmbedReq>({
    mutationFn: (req) => api.putApiGamesIdEmbed(gameId, req).then((r) => r.data.embed ?? null),
    onSuccess: (embed) => queryClient.setQueryData(queryKeys.gameEmbed(gameId), embed),
    onError: handleApiError,
  });
}
