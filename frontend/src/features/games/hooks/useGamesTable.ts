import {
  useCallback,
  useEffect,
  useRef,
  useState,
  type ChangeEvent,
  type RefObject,
} from "react";
import { useNavigate } from "@tanstack/react-router";
import {
  useCreateGame,
  useDeleteGame,
  useExportGameYaml,
  useFetchGame,
  useGame,
  useGames,
} from "@/api/hooks";
import type { CreateGameReq, GameView } from "@/api/types";
import { useAuth } from "@/providers/AuthProvider";
import type { CreateGameFormData } from "../types";
import {
  downloadYamlFile,
  gameToFormData,
  parseGameYaml,
} from "../lib";
import { GAME_LIST_PRESETS, type GameListPreset } from "../lib/gameListPresets";
import type { GameEditRights } from "../components/GameEditModal";
import { useFavoriteState } from "./useFavoriteState";
import { useGameNavigation } from "./useGameNavigation";
import { useGameSessionState } from "./useGameSessionState";

export interface UseGamesTableOptions {
  preset: GameListPreset;
  /** Open this game's modal on mount (a /games/<id> link). */
  initialGameId?: string;
  /** Open the create modal on mount. */
  initialCreate?: boolean;
  /** Open the import file dialog on mount. */
  autoImport?: boolean;
  /** Called when a modal opened on mount is closed. */
  onModalClose?: () => void;
}

interface UseGamesTableDeps {
  /** The hidden file input the import opens. */
  fileInputRef: RefObject<HTMLInputElement | null>;
}

/**
 * What the caller may do with a game: the server's flags on it, read as they
 * are. Nothing here works a right out; the server enforces every action.
 */
export interface GamePermissions {
  /** A fact, for presentation (badges, warnings), never for rights. */
  isOwner: boolean;
  rights: GameEditRights;
  canEdit: boolean;
  /** Deleting is part of editing the settings. */
  canDelete: boolean;
  canCopy: boolean;
}


/**
 * Everything a game list does, for GamesTable to render: the server-side
 * query and page, what the caller may do with each game, and the modals for
 * viewing, creating, copying, importing, sponsoring and deleting.
 */
export function useGamesTable(options: UseGamesTableOptions, { fileInputRef }: UseGamesTableDeps) {
  const { preset, initialGameId, initialCreate, autoImport, onModalClose } = options;
  const config = GAME_LIST_PRESETS[preset];
  const navigate = useNavigate();
  const { backendUser } = useAuth();
  const userId = backendUser?.id;
  const workshopId = preset === "workshop" ? backendUser?.role?.workshop?.id : undefined;

  // ── The list ────────────────────────────────────────────────────────────
  const list = useGames(config.list);

  // ── Who may do what: the server's flags ────────────────────────────────
  /** The game belongs to the workshop the caller is in (a fact, for badges). */
  const isWorkshopGame = useCallback(
    (game: Pick<GameView, "workshopId">) => !!workshopId && game.workshopId === workshopId,
    [workshopId],
  );
  const permissions = useCallback(
    (game: GameView): GamePermissions => {
      const rights = {
        card: game.mayEditCard,
        content: game.mayEditContent,
        settings: game.mayEditSettings,
      };
      return {
        isOwner: !!userId && game.creatorId === userId,
        rights,
        canEdit: rights.card || rights.content || rights.settings,
        canDelete: game.mayDelete,
        canCopy: game.mayCopy,
      };
    },
    [userId],
  );

  // ── Playing and favourites ──────────────────────────────────────────────
  const { sessionsLoading, getSessionState } = useGameSessionState();
  const { playGame, continueGame, restartGame } = useGameNavigation();
  const { isFavorite, toggleFavorite } = useFavoriteState();

  // ── Viewing and editing one game ────────────────────────────────────────
  // The game comes from the server with the caller's rights on it, also when
  // it is opened by link (/games/<id>).
  const [viewingId, setViewingId] = useState<string | null>(initialGameId ?? null);
  const openedOnMount = useRef(!!initialGameId || !!initialCreate);
  const { data: viewedGame } = useGame(viewingId ?? undefined);
  const viewedPermissions = viewedGame ? permissions(viewedGame) : null;

  const view = useCallback((game: GameView) => setViewingId(game.id), []);

  const closeMountModal = () => {
    if (openedOnMount.current) {
      openedOnMount.current = false;
      onModalClose?.();
    }
  };

  const closeView = () => {
    setViewingId(null);
    closeMountModal();
  };

  // ── Creating, copying, importing ────────────────────────────────────────
  const createGame = useCreateGame();
  const fetchGame = useFetchGame();
  const [creating, setCreating] = useState<{ initialData: Partial<CreateGameFormData> | null } | null>(
    initialCreate ? { initialData: null } : null,
  );

  const openCreate = () => setCreating({ initialData: null });
  const closeCreate = () => {
    setCreating(null);
    closeMountModal();
  };

  const submitCreate = async (request: CreateGameReq) => {
    try {
      const newGame = await createGame.mutateAsync(request);
      setCreating(null);
      if (config.openCreatedGame && newGame.id) {
        navigate({ to: "/games/$gameId", params: { gameId: newGame.id } });
      }
      return newGame;
    } catch {
      // shown by the mutation's error handler
      return undefined;
    }
  };

  /** Copies a game, loaded with all the caller may read of it. */
  const copy = async (gameId: string) => {
    const full = await fetchGame(gameId);
    setViewingId(null);
    setCreating({ initialData: gameToFormData(full) });
  };

  const openImport = () => fileInputRef.current?.click();

  const importFile = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;
    file
      .text()
      .then((content) => setCreating({ initialData: parseGameYaml(content) }))
      .catch(() => {
        // an unreadable file opens nothing
      });
  };

  useEffect(() => {
    if (!autoImport) return;
    const timer = setTimeout(() => fileInputRef.current?.click(), 100);
    return () => clearTimeout(timer);
  }, [autoImport, fileInputRef]);

  // ── Exporting ───────────────────────────────────────────────────────────
  const exportYaml = useExportGameYaml();
  const exportGame = async (game: GameView) => {
    try {
      downloadYamlFile(await exportYaml.mutateAsync(game.id), game.name);
    } catch {
      // shown by the mutation's error handler
    }
  };

  // ── Deleting ────────────────────────────────────────────────────────────
  const deleteGame = useDeleteGame();
  const [deleting, setDeleting] = useState<GameView | null>(null);
  const confirmDelete = async () => {
    if (!deleting) return;
    try {
      await deleteGame.mutateAsync(deleting.id);
      setDeleting(null);
    } catch {
      // shown by the mutation's error handler
    }
  };

  // ── Sponsoring ──────────────────────────────────────────────────────────
  const [sponsoring, setSponsoring] = useState<GameView | null>(null);

  return {
    config,
    list: { ...list, isInitialLoading: list.isInitialLoading || sessionsLoading },
    permissions,
    isWorkshopGame,
    isFavorite,
    toggleFavorite,
    getSessionState,
    playGame,
    continueGame,
    restartGame,
    view,
    copy,
    exportGame,
    requestDelete: setDeleting,
    openCreate,
    openImport,
    importFile,
    viewModal: {
      gameId: viewingId,
      opened: !!viewingId,
      close: closeView,
      isOwner: viewedPermissions?.isOwner ?? false,
      onCopy:
        viewingId && viewedPermissions && !viewedPermissions.canEdit && viewedPermissions.canCopy
          ? () => copy(viewingId)
          : undefined,
      // Sponsoring sets the game's key: a settings change. Workshop lists
      // don't offer it (the workshop's key plays there).
      onSponsor:
        viewedPermissions?.rights.settings && preset !== "workshop" && viewedGame
          ? () => setSponsoring(viewedGame)
          : undefined,
      workshopId,
    },
    createModal: {
      opened: !!creating,
      initialData: creating?.initialData ?? null,
      close: closeCreate,
      submit: submitCreate,
      loading: createGame.isPending,
    },
    deleteModal: {
      game: deleting,
      opened: !!deleting,
      close: () => setDeleting(null),
      confirm: confirmDelete,
      loading: deleteGame.isPending,
      isOwner: deleting ? permissions(deleting).isOwner : false,
      /** Someone else's game of the workshop: the dialog says so (a fact). */
      isOthersWorkshopGame:
        !!deleting && !permissions(deleting).isOwner && isWorkshopGame(deleting),
    },
    sponsorModal: {
      game: sponsoring,
      opened: !!sponsoring,
      close: () => setSponsoring(null),
    },
  };
}

export type GamesTableState = ReturnType<typeof useGamesTable>;
