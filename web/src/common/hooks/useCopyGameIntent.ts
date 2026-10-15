import { useState } from "react";
import { useDisclosure } from "@mantine/hooks";
import { useNavigate } from "@tanstack/react-router";
import { useAuthenticatedApi } from "@/api/useAuthenticatedApi";
import { useCreateGame, useUpdateGame } from "@/api/hooks";
import { useAuth } from "@/providers/AuthProvider";
import { ROUTES } from "@/common/routes/routes";
import {
  rememberCopyIntent,
  rememberReturnTo,
  takeCopyIntent,
  takeReturnTo,
} from "@/common/lib/returnTo";
import {
  createGameWithExtraFields,
  gameToFormData,
} from "@/features/games/lib";
import type { CreateGameFormData } from "@/features/games/types";

/**
 * Copying a public game from a page anyone may open — the public workshop page
 * or a game's share link. A visitor without an account is sent through login,
 * and registration if needed; `scope` is the path they come back to, and the
 * game they picked waits in sessionStorage until then.
 */
export function useCopyGameIntent(scope: string) {
  const navigate = useNavigate();
  const { isAuthenticated, isParticipant, backendUser } = useAuth();
  const authApi = useAuthenticatedApi();
  const createGame = useCreateGame();
  const updateGame = useUpdateGame();
  const [copyData, setCopyData] = useState<Partial<CreateGameFormData> | null>(
    null,
  );
  const [opened, { open, close }] = useDisclosure(false);

  /** True once an authenticated visitor can be asked to copy. */
  const ready = !isAuthenticated || !!(backendUser && authApi);

  const start = async (gameId: string) => {
    if (!isAuthenticated || !authApi) {
      rememberReturnTo(scope);
      rememberCopyIntent({ scope, gameId });
      navigate({ to: ROUTES.AUTH_LOGIN });
      return;
    }
    const full = (await authApi.games.gamesDetail(gameId)).data;
    setCopyData(gameToFormData(full));
    open();
  };

  /** Picks up the game remembered before login. Call once the page has loaded. */
  const resume = () => {
    // The visitor is back, so the remembered page has done its job.
    takeReturnTo();
    const gameId = takeCopyIntent(scope);
    if (gameId) void start(gameId);
  };

  const createCopy = async (data: CreateGameFormData) => {
    const newGame = await createGameWithExtraFields(
      data,
      createGame.mutateAsync,
      updateGame.mutateAsync,
    );
    close();
    setCopyData(null);
    if (isParticipant) {
      navigate({ to: ROUTES.MY_WORKSHOP as "/" });
    } else if (newGame.id) {
      navigate({ to: `/my-games/${newGame.id}` as "/" });
    }
  };

  return {
    /** Starts a copy, or sends the visitor to log in first. */
    start,
    resume,
    ready,
    modal: {
      opened,
      close: () => {
        close();
        setCopyData(null);
      },
      initialData: copyData,
      onCreate: createCopy,
      createLoading: createGame.isPending,
    },
  };
}
