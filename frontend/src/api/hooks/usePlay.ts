import { useCallback } from "react";
import { useMutation, useQueryClient } from "@tanstack/react-query";
import { api } from "../client/api";
import { queryKeys } from "../queryKeys";
import type { CreateRequest, GuestGameInfoResp, Mode } from "../generated/api.gen";

/**
 * A session to create: a game's, its prompts adapted where prompts says, or a
 * share link's. A mode, live or rounds, is the player's wish for a game that
 * plays both.
 */
export type NewSession =
  | { readonly game: string; readonly prompts?: Record<string, string>; readonly mode?: Mode }
  | { readonly share: string; readonly mode?: Mode };

const ONCE_PREFIX = "cgl.once.";

/**
 * Creates an engine session, which replaces the player's earlier sessions
 * of the game. Its one-time key is kept for this tab until the session
 * exists, so a reload while creating answers the same session rather than
 * starting a second; a refusal lets the key go, so trying again is a new
 * intent. Throws as the generated client does.
 */
export function useCreateSession() {
  const queryClient = useQueryClient();
  return useMutation<string, unknown, { what: NewSession; language: string }>({
    mutationFn: async ({ what, language }) => {
      const slot =
        ONCE_PREFIX +
        ("game" in what ? `game:${what.game}` : `share:${what.share}`);
      const once = tabStorage.get(slot) ?? randomKey();
      tabStorage.set(slot, once);
      try {
        const request: CreateRequest =
          "game" in what
            ? { game: what.game, promptOverride: what.prompts, language, once, mode: what.mode }
            : { share: what.share, language, once, mode: what.mode };
        const { data } = await api.postApiEngineSessions(request);
        tabStorage.remove(slot);
        return data.id;
      } catch (thrown) {
        // Answered with a refusal: the next try is a new intent.
        if (thrown instanceof Response) tabStorage.remove(slot);
        throw thrown;
      }
    },
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.userSessions });
      queryClient.invalidateQueries({ queryKey: queryKeys.gameSessions });
      queryClient.invalidateQueries({ queryKey: queryKeys.games });
    },
  });
}

/**
 * Opens an engine session: the engine restores it if it is not running,
 * and refuses if the caller may not play it now. Throws as the generated
 * client does.
 */
export function useOpenSession() {
  return useMutation<string, unknown, string>({
    mutationFn: async (id) => {
      await api.getApiEngineSessionsIdTopology(id);
      return id;
    },
  });
}

/**
 * Reads what a share link offers now (its game, plays left, whether a new
 * session can start), fresh on every call: plays are used up by others.
 */
export function useFetchShareInfo() {
  const queryClient = useQueryClient();
  return useCallback(
    (token: string) =>
      queryClient.fetchQuery<GuestGameInfoResp>({
        queryKey: queryKeys.shareInfo(token),
        queryFn: () => api.getApiPlayTokenInfo(token).then((r) => r.data),
        staleTime: 0,
      }),
    [queryClient],
  );
}

function randomKey(): string {
  const bytes = new Uint8Array(16);
  crypto.getRandomValues(bytes);
  return Array.from(bytes, (b) => b.toString(16).padStart(2, "0")).join("");
}

/** This tab's storage, which a private window or a blocked site may refuse: then nothing is kept. */
const tabStorage = {
  get(key: string): string | null {
    try {
      return sessionStorage.getItem(key);
    } catch {
      return null;
    }
  },
  set(key: string, value: string): void {
    try {
      sessionStorage.setItem(key, value);
    } catch {
      // Kept nowhere: a reload mid-creation may start a second session.
    }
  },
  remove(key: string): void {
    try {
      sessionStorage.removeItem(key);
    } catch {
      // Nothing kept, nothing to remove.
    }
  },
};
