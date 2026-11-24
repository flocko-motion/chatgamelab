import { useCallback, useEffect, useRef, useState } from "react";
import { useDeleteGame, useGame, useGameTemplate, useUpdateGame } from "@/api/hooks";
import { useAdmin } from "@/common/hooks/useAdmin";
import type { CreateGameReq, GameTemplate, GameView } from "@/api/types";
import { gameChanges, gameCreate, type GameFormFields } from "../lib/gameUpdate";
import type { CreateGameFormData } from "../types";
import { useDrawCoverOnSave } from "./useGameCover";

/** Games without a genre are adventures. */
export const DEFAULT_GENRE = "adventure";

/** The longest game name. */
export const NAME_MAX = 70;

/** The sections of a game the caller may change; the server decides. */
export interface GameEditRights {
  card: boolean;
  content: boolean;
  settings: boolean;
}

const EMPTY: GameFormFields = {
  name: "",
  description: "",
  public: false,
  systemMessageScenario: "",
  systemMessageGameStart: "",
  imageStyle: "",
  voiceStyle: "",
  voices: {},
  statusFields: "",
  genre: DEFAULT_GENRE,
  theme: "",
};

function fromGame(g: GameView | GameTemplate): GameFormFields {
  return {
    name: g.name ?? "",
    description: g.description ?? "",
    public: g.public ?? false,
    systemMessageScenario: g.systemMessageScenario ?? "",
    systemMessageGameStart: g.systemMessageGameStart ?? "",
    imageStyle: g.imageStyle ?? "",
    voiceStyle: g.voiceStyle ?? "",
    voices: g.voices ?? {},
    statusFields: g.statusFields ?? "",
    genre: g.genre || DEFAULT_GENRE,
    theme: g.theme ?? "",
  };
}

/** The template, with what an import or a copy brings. */
function withInitialData(base: GameFormFields, data: Partial<CreateGameFormData> | null | undefined): GameFormFields {
  if (!data) return base;
  return {
    name: data.name ?? base.name,
    description: data.description ?? base.description,
    public: data.isPublic ?? base.public,
    systemMessageScenario: data.systemMessageScenario ?? base.systemMessageScenario,
    systemMessageGameStart: data.systemMessageGameStart ?? base.systemMessageGameStart,
    imageStyle: data.imageStyle ?? base.imageStyle,
    voiceStyle: data.voiceStyle ?? base.voiceStyle,
    voices: data.voices ?? base.voices,
    statusFields: data.statusFields ?? base.statusFields,
    genre: data.genre || base.genre,
    theme: data.theme ?? base.theme,
  };
}

/** The fields as they are sent: text trimmed. */
function trimmed(v: GameFormFields): GameFormFields {
  return {
    ...v,
    name: v.name.trim(),
    description: v.description.trim(),
    systemMessageScenario: v.systemMessageScenario.trim(),
    systemMessageGameStart: v.systemMessageGameStart.trim(),
    imageStyle: v.imageStyle.trim(),
    voiceStyle: v.voiceStyle.trim(),
    statusFields: v.statusFields.trim(),
  };
}

interface Options {
  /** The game to edit; none makes a new one. */
  gameId?: string | null;
  opened: boolean;
  /** Pre-populates a new game (an import or a copy). */
  initialData?: Partial<CreateGameFormData> | null;
  /** Creates the game; answers with it as created, so its cover is drawn. */
  onCreate?: (request: CreateGameReq) => Promise<GameView | undefined> | void;
  /** The editor is done: saved, or nothing to save. */
  onDone: () => void;
  /** The caller made the game (a fact, for the admin warning). */
  isOwner?: boolean;
}

/**
 * The game editor's form: the game or the template it starts from, the
 * fields as edited, what the caller may change, and saving. A new game
 * sends what differs from the template; an edit sends what changed.
 */
export function useGameEditor({ gameId, opened, initialData, onCreate, onDone, isOwner = true }: Options) {
  const { isAdmin } = useAdmin();
  const creating = !gameId;
  const gameQuery = useGame(gameId ?? "");
  const templateQuery = useGameTemplate(creating && opened);
  const game = gameQuery.data;
  const template = templateQuery.data;
  const loading = creating ? templateQuery.isLoading : gameQuery.isLoading;
  const failed = !!(creating ? templateQuery.error : gameQuery.error) || (!loading && !(creating ? template : game));

  const updateGame = useUpdateGame();
  const drawCoverOnSave = useDrawCoverOnSave();

  const [values, setValues] = useState<GameFormFields>(EMPTY);
  const [nameError, setNameError] = useState<string | null>(null);
  // What the form started from: dirty checks and updates compare with it.
  const [loaded, setLoaded] = useState<GameFormFields | null>(null);
  // A new game sends what differs from the template.
  const templateValues = useRef<GameFormFields | null>(null);
  // What the author changed before the game or template arrived: it stays
  // when the form starts from them, so nothing picked early is overwritten.
  const touched = useRef<Partial<GameFormFields>>({});

  /* eslint-disable react-hooks/set-state-in-effect -- Intentional: the form starts from the loaded game */
  useEffect(() => {
    if (!opened) {
      setLoaded(null);
      templateValues.current = null;
      touched.current = {};
      setValues(EMPTY);
      setNameError(null);
      return;
    }
    if (loaded) return;
    let start: GameFormFields | null = null;
    if (!creating && game) {
      start = fromGame(game);
    } else if (creating && template) {
      templateValues.current = fromGame(template);
      start = withInitialData(templateValues.current, initialData);
    }
    if (start) {
      setLoaded(start);
      setValues({ ...start, ...touched.current });
    }
  }, [opened, creating, game, template, initialData, loaded]);
  /* eslint-enable react-hooks/set-state-in-effect */

  const set = useCallback(
    <K extends keyof GameFormFields>(field: K, value: GameFormFields[K]) => {
      if (!loaded) touched.current = { ...touched.current, [field]: value };
      setValues((v) => ({ ...v, [field]: value }));
      if (field === "name") setNameError(null);
    },
    [loaded],
  );

  const dirty = !!loaded && Object.keys(gameChanges(loaded, values)).length > 0;

  // A new game shows its template's rights; it has no settings yet.
  const rights: GameEditRights = creating
    ? { card: !!template?.mayEditCard, content: !!template?.mayEditContent, settings: false }
    : { card: !!game?.mayEditCard, content: !!game?.mayEditContent, settings: !!game?.mayEditSettings };
  const readOnly = !rights.card && !rights.content && !rights.settings;

  // Switching public on needs mayPublish, switching a public game off a card
  // change; taking back an unsaved switch is always possible.
  const wasPublic = !!game?.public;
  const publicSwitch = {
    shown: !!game?.mayPublish || rights.card,
    enabled: values.public ? rights.card || !wasPublic : !!game?.mayPublish,
  };

  /** Whether the name will do; says why not, as a translation key. */
  const checkName = useCallback((): boolean => {
    const name = values.name.trim();
    if (!name) setNameError("games.errors.nameRequired");
    else if (name.length > NAME_MAX) setNameError("games.errors.nameTooLong");
    else return true;
    return false;
  }, [values.name]);

  const deleteGame = useDeleteGame();
  /** Deletes the game for good; the editor is done. */
  const remove = async () => {
    if (!game?.id) return;
    try {
      await deleteGame.mutateAsync(game.id);
      onDone();
    } catch {
      // The mutation shows the error.
    }
  };

  const save = async () => {
    if (!checkName()) return;
    const edited = trimmed(values);
    if (creating) {
      if (!templateValues.current) return;
      drawCoverOnSave((await onCreate?.(gameCreate(templateValues.current, edited))) ?? undefined);
      return;
    }
    if (!game?.id || !loaded) return;
    const changes = gameChanges(loaded, edited);
    if (Object.keys(changes).length === 0) {
      drawCoverOnSave(game);
      onDone();
      return;
    }
    try {
      drawCoverOnSave(await updateGame.mutateAsync({ id: game.id, changes }));
      onDone();
    } catch {
      // The mutation shows the error.
    }
  };

  return {
    creating,
    /** A new game from nothing: no copy, no import. It starts with the genre. */
    fresh: creating && !initialData,
    game,
    template,
    loading,
    failed,
    values,
    set,
    dirty,
    rights,
    readOnly,
    /** Whether the public switch shows, and may be switched now. */
    publicSwitch,
    /** An admin opened a game that is not theirs (a fact, for the warning). */
    adminAction: isAdmin && !creating && !isOwner,
    /** A translation key when the name will not do. */
    nameError,
    checkName,
    save,
    saving: !creating && updateGame.isPending,
    remove,
    removing: deleteGame.isPending,
  };
}

export type GameEditor = ReturnType<typeof useGameEditor>;
