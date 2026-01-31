import { useMemo, useRef, useCallback } from "react";
import { useDebouncedValue } from "@mantine/hooks";
import {
  useGames,
  useCreateGame,
  useUpdateGame,
  useDeleteGame,
  useExportGameYaml,
  useImportGameYaml,
  useGameSessionMap,
  useDeleteSession,
  useCloneGame,
} from "@/api/hooks";
import type { ObjGame } from "@/api/generated";
import type { CreateGameFormData } from "@/features/games/types";
import {
  type GameFilter,
  type WorkshopSettings,
  type GameSessionState,
  filterGamesByWorkshopSettings,
  filterGamesByUserFilter,
  getGamePermissions,
} from "../types";

interface UseWorkshopGamesOptions {
  currentUserId: string | undefined;
  canEditAllWorkshopGames: boolean;
  workshopSettings: WorkshopSettings;
  gameFilter: GameFilter;
  sortValue: string;
  searchQuery: string;
}

export function useWorkshopGames(options: UseWorkshopGamesOptions) {
  const {
    currentUserId,
    canEditAllWorkshopGames,
    workshopSettings,
    gameFilter,
    sortValue,
    searchQuery,
  } = options;

  const fileInputRef = useRef<HTMLInputElement>(null);
  const [debouncedSearch] = useDebouncedValue(searchQuery, 300);
  const [sortField, sortDir] = sortValue.split("-") as [string, "asc" | "desc"];

  // Fetch games
  const {
    data: rawGames,
    isLoading,
    isFetching,
    error,
    refetch,
  } = useGames({
    sortBy: sortField as "name" | "createdAt" | "modifiedAt" | "playCount" | "visibility" | "creator",
    sortDir,
    filter: "all",
    search: debouncedSearch || undefined,
  });

  const { sessionMap, isLoading: sessionsLoading } = useGameSessionMap();
  const createGame = useCreateGame();
  const updateGame = useUpdateGame();
  const deleteGame = useDeleteGame();
  const deleteSession = useDeleteSession();
  const exportGameYaml = useExportGameYaml();
  const importGameYaml = useImportGameYaml();
  const cloneGame = useCloneGame();

  // Apply filters
  const games = useMemo(() => {
    if (!rawGames) return [];
    const settingsFiltered = filterGamesByWorkshopSettings(
      rawGames,
      currentUserId,
      workshopSettings,
      canEditAllWorkshopGames,
    );
    return filterGamesByUserFilter(settingsFiltered, gameFilter, currentUserId);
  }, [rawGames, gameFilter, currentUserId, canEditAllWorkshopGames, workshopSettings]);

  // Permission helpers
  const getPermissions = useCallback(
    (game: ObjGame) => getGamePermissions(game, currentUserId, canEditAllWorkshopGames),
    [currentUserId, canEditAllWorkshopGames],
  );

  // Session helpers
  const getSessionState = useCallback(
    (game: ObjGame): GameSessionState => {
      if (!game.id) return { hasSession: false, session: undefined };
      const session = sessionMap.get(game.id);
      return { hasSession: !!session, session };
    },
    [sessionMap],
  );

  // Game operations
  const handleCreateGame = useCallback(
    async (data: CreateGameFormData) => {
      const newGame = await createGame.mutateAsync({
        name: data.name,
        description: data.description,
        public: data.isPublic,
      });

      const hasExtraFields =
        data.systemMessageScenario ||
        data.systemMessageGameStart ||
        data.imageStyle ||
        data.statusFields;

      if (newGame.id && hasExtraFields) {
        await updateGame.mutateAsync({
          id: newGame.id,
          game: {
            ...newGame,
            systemMessageScenario: data.systemMessageScenario,
            systemMessageGameStart: data.systemMessageGameStart,
            imageStyle: data.imageStyle,
            statusFields: data.statusFields,
          },
        });
      }

      return newGame;
    },
    [createGame, updateGame],
  );

  const handleDeleteGame = useCallback(
    async (gameId: string) => {
      await deleteGame.mutateAsync(gameId);
    },
    [deleteGame],
  );

  const handleExportGame = useCallback(
    async (game: ObjGame) => {
      if (!game.id) return;
      const yaml = await exportGameYaml.mutateAsync(game.id);
      const blob = new Blob([yaml], { type: "application/x-yaml" });
      const url = URL.createObjectURL(blob);
      const link = document.createElement("a");
      link.href = url;
      link.download = `${game.name || "game"}.yaml`;
      document.body.appendChild(link);
      link.click();
      document.body.removeChild(link);
      URL.revokeObjectURL(url);
    },
    [exportGameYaml],
  );

  const handleCloneGame = useCallback(
    async (gameId: string) => {
      return await cloneGame.mutateAsync(gameId);
    },
    [cloneGame],
  );

  const handleDeleteSession = useCallback(
    async (sessionId: string) => {
      await deleteSession.mutateAsync(sessionId);
    },
    [deleteSession],
  );

  const triggerImportClick = useCallback(() => {
    fileInputRef.current?.click();
  }, []);

  const handleImportFile = useCallback(
    async (file: File): Promise<string | undefined> => {
      return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = async (e) => {
          const content = e.target?.result as string;
          let newGameId: string | undefined;

          try {
            const nameMatch = content.match(/^name:\s*["']?(.+?)["']?\s*$/m);
            const gameName = nameMatch?.[1]?.trim() || file.name.replace(/\.(yaml|yml)$/i, "");

            const newGame = await createGame.mutateAsync({ name: gameName });
            newGameId = newGame.id;

            if (newGameId) {
              await importGameYaml.mutateAsync({ id: newGameId, yaml: content });
              refetch();
              resolve(newGameId);
            }
          } catch (err) {
            if (newGameId) {
              try {
                await deleteGame.mutateAsync(newGameId);
                refetch();
              } catch {
                // Ignore cleanup errors
              }
            }
            reject(err);
          }
        };
        reader.readAsText(file);
      });
    },
    [createGame, importGameYaml, deleteGame, refetch],
  );

  return {
    // Data
    games,
    rawGames,
    sortField,

    // Loading states
    isLoading,
    isFetching,
    sessionsLoading,
    error,

    // Mutation states
    isCreating: createGame.isPending,
    isDeleting: deleteGame.isPending,
    isCloning: cloneGame.isPending,

    // Helpers
    getPermissions,
    getSessionState,
    fileInputRef,

    // Operations
    handleCreateGame,
    handleDeleteGame,
    handleExportGame,
    handleCloneGame,
    handleDeleteSession,
    triggerImportClick,
    handleImportFile,
  };
}
