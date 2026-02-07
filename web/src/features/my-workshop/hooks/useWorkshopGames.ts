import { useMemo, useRef, useCallback } from "react";
import { useDebouncedValue } from "@mantine/hooks";
import {
  useGames,
  useCreateGame,
  useUpdateGame,
  useDeleteGame,
  useExportGameYaml,
  useGameSessionMap,
  useDeleteSession,
} from "@/api/hooks";
import type { ObjGame } from "@/api/generated";
import type { CreateGameFormData } from "@/features/games/types";
import { parseGameYaml, gameToFormData } from "@/features/games/lib";
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
  currentWorkshopId: string | undefined;
  canEditAllWorkshopGames: boolean;
  workshopSettings: WorkshopSettings;
  gameFilter: GameFilter;
  sortValue: string;
  searchQuery: string;
}

export function useWorkshopGames(options: UseWorkshopGamesOptions) {
  const {
    currentUserId,
    currentWorkshopId,
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
    sortBy: sortField as
      | "name"
      | "createdAt"
      | "modifiedAt"
      | "playCount"
      | "visibility"
      | "creator",
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

  // Apply filters
  const games = useMemo(() => {
    if (!rawGames) return [];
    const settingsFiltered = filterGamesByWorkshopSettings(
      rawGames,
      currentUserId,
      workshopSettings,
      canEditAllWorkshopGames,
    );
    return filterGamesByUserFilter(
      settingsFiltered,
      gameFilter,
      currentUserId,
      currentWorkshopId,
    );
  }, [
    rawGames,
    gameFilter,
    currentUserId,
    currentWorkshopId,
    canEditAllWorkshopGames,
    workshopSettings,
  ]);

  // Permission helpers
  const getPermissions = useCallback(
    (game: ObjGame) =>
      getGamePermissions(game, currentUserId, canEditAllWorkshopGames),
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

  const getGameFormDataForCopy = useCallback(
    (gameId: string): Partial<CreateGameFormData> | null => {
      const game = rawGames?.find((g) => g.id === gameId);
      if (!game) return null;
      return gameToFormData(game);
    },
    [rawGames],
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

  const parseImportFile = useCallback(
    (file: File): Promise<Partial<CreateGameFormData>> => {
      return new Promise((resolve, reject) => {
        const reader = new FileReader();
        reader.onload = (e) => {
          try {
            const content = e.target?.result as string;
            resolve(parseGameYaml(content));
          } catch (err) {
            reject(err);
          }
        };
        reader.onerror = () => reject(new Error("Failed to read file"));
        reader.readAsText(file);
      });
    },
    [],
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

    // Helpers
    getPermissions,
    getSessionState,
    fileInputRef,

    // Operations
    handleCreateGame,
    handleDeleteGame,
    handleExportGame,
    getGameFormDataForCopy,
    handleDeleteSession,
    triggerImportClick,
    parseImportFile,

    // Refetch (for SSE events)
    refetchGames: refetch,
  };
}
