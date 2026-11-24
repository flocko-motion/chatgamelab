import {
  useCallback,
  useEffect,
  useMemo,
  useRef,
  useState,
  type ChangeEvent,
  type RefObject,
} from "react";
import { useNavigate } from "@tanstack/react-router";
import { notifications } from "@mantine/notifications";
import { useTranslation } from "react-i18next";
import {
  useCreateGame,
  useFetchGame,
  useGame,
  useGames,
  useGenres,
  useImportGameCover,
} from "@/api/hooks";
import { extractRawErrorCode } from "@/common/types/errorCodes";
import type { CreateGameReq, GameView } from "@/api/types";
import { useAuth } from "@/providers/AuthProvider";
import { isOrganisationMember } from "@/common/lib/roles";
import type { CreateGameFormData } from "../types";
import {
  gameToFormData,
  GameFileError,
  readGameFile,
  type ImportedCover,
} from "../lib";
import {
  GAME_LIST_PRESETS,
  withGenreFilter,
  withoutOrganisationFilter,
  type GameListPreset,
} from "../lib/gameListPresets";
import type { GameEditRights } from "./useGameEditor";
import { useFavoriteState } from "./useFavoriteState";
import { useGameNavigation } from "@/common/hooks/useGameNavigation";
import { useGameSessionState } from "./useGameSessionState";

export interface UseGamesTableOptions {
  preset: GameListPreset;
  /** Open this game's modal on mount (a /games/<id> link). */
  initialGameId?: string;
  /** Open the create modal on mount. */
  initialCreate?: boolean;
  /** What the new game is about, filled in when the create modal opens on mount. */
  initialAbout?: string;
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
 * query and page, what the caller may do with each game, and the editor and
 * modals for viewing, creating, copying, importing and sponsoring. Exporting
 * and deleting live in the editor.
 */
export function useGamesTable(options: UseGamesTableOptions, { fileInputRef }: UseGamesTableDeps) {
  const { preset, initialGameId, initialCreate, initialAbout, autoImport, onModalClose } = options;
  const config = GAME_LIST_PRESETS[preset];
  const navigate = useNavigate();
  const { t } = useTranslation("common");
  const { backendUser } = useAuth();
  const userId = backendUser?.id;
  const workshopId = preset === "workshop" ? backendUser?.role?.workshop?.id : undefined;

  // ── The list ────────────────────────────────────────────────────────────
  // The gallery can be narrowed to genres, the ones the engine plays, and,
  // for the head and staff of an organisation, to its games.
  const { data: genres } = useGenres();
  const genreIds = useMemo(() => (genres ?? []).map((g) => g.id), [genres]);
  const inOrganisation = isOrganisationMember(backendUser);
  const listPreset = useMemo(() => {
    if (preset !== "all") return config.list;
    const list = inOrganisation ? config.list : withoutOrganisationFilter(config.list);
    return withGenreFilter(list, genreIds);
  }, [preset, config.list, genreIds, inOrganisation]);
  const list = useGames(listPreset);

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
  const { playGame, continueGame, restartGame, openGame } = useGameNavigation();
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
  // copy: the form holds someone else's game, which saving makes the caller's own.
  // cover: the picture an imported .cgl file brought along, made the new game's cover once it exists.
  const [creating, setCreating] = useState<{
    initialData: Partial<CreateGameFormData> | null;
    copy?: boolean;
    cover?: ImportedCover;
  } | null>(
    initialCreate
      ? { initialData: initialAbout ? { systemMessageScenario: initialAbout } : null }
      : null,
  );
  const importCover = useImportGameCover();

  /**
   * Makes an imported cover the new game's, and answers the game as it then
   * is. A cover the server refuses leaves the game without one, and says why;
   * the editor then draws one as for any new game.
   */
  const withImportedCover = async (game: GameView, cover: ImportedCover | undefined): Promise<GameView> => {
    if (!cover || !game.id) return game;
    try {
      return await importCover.mutateAsync({ id: game.id, ...cover });
    } catch (error) {
      const code = extractRawErrorCode(error);
      notifications.show({
        color: "yellow",
        message: t(
          code === "cover_refused"
            ? "games.importExport.coverRefused"
            : code === "cover_unsigned"
              ? "games.importExport.coverUnsigned"
              : code === "no_api_key"
              ? "games.importExport.coverNoKey"
              : "games.importExport.coverFailed",
        ),
      });
      return game;
    }
  };

  const openCreate = () => setCreating({ initialData: null });
  const closeCreate = () => {
    setCreating(null);
    closeMountModal();
  };

  const submitCreate = async (request: CreateGameReq) => {
    try {
      const cover = creating?.cover;
      const newGame = await withImportedCover(await createGame.mutateAsync(request), cover);
      const wasCopy = !!creating?.copy;
      setCreating(null);
      if (wasCopy) {
        notifications.show({ message: t("games.editModal.savedAsOwn"), color: "green" });
      }
      if (config.openCreatedGame && newGame.id) {
        navigate({ to: "/games/$gameId", params: { gameId: newGame.id } });
      } else if (wasCopy && newGame.id) {
        // The editor goes on with the game as the caller's own now.
        setViewingId(newGame.id);
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
    setCreating({ initialData: gameToFormData(full), copy: true });
  };

  /**
   * Opens a game in the editor as itself: to change when the caller may, to
   * look at otherwise, with a copy of their own a button away.
   */
  const open = view;

  const openImport = () => fileInputRef.current?.click();

  const importFile = (event: ChangeEvent<HTMLInputElement>) => {
    const file = event.target.files?.[0];
    event.target.value = "";
    if (!file) return;
    readGameFile(file)
      .then(({ data, cover }) => setCreating({ initialData: data, cover }))
      .catch((error: unknown) => {
        // an unreadable file opens nothing, and says why
        notifications.show({
          color: "red",
          message: t(error instanceof GameFileError ? error.key : "games.importExport.errors.unreadable"),
        });
      });
  };

  useEffect(() => {
    if (!autoImport) return;
    const timer = setTimeout(() => fileInputRef.current?.click(), 100);
    return () => clearTimeout(timer);
  }, [autoImport, fileInputRef]);

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
    openGame,
    continueGame,
    restartGame,
    view: open,
    copy,
    openCreate,
    openImport,
    importFile,
    viewModal: {
      gameId: viewingId,
      opened: !!viewingId,
      close: closeView,
      isOwner: viewedPermissions?.isOwner ?? false,
      /** Someone else's game of the workshop: the delete page says so (a fact). */
      isOthersWorkshopGame:
        !!viewedGame && !!viewedPermissions && !viewedPermissions.isOwner && isWorkshopGame(viewedGame),
      /** Makes the caller's own copy of the game they are looking at. */
      copy: () => {
        if (viewingId) void copy(viewingId);
      },
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
      saveAsOwn: !!creating?.copy,
      close: closeCreate,
      submit: submitCreate,
      loading: createGame.isPending || importCover.isPending,
    },
    sponsorModal: {
      game: sponsoring,
      opened: !!sponsoring,
      close: () => setSponsoring(null),
    },
  };
}

export type GamesTableState = ReturnType<typeof useGamesTable>;
