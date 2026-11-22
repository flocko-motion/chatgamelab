import {
  Modal,
  Stack,
  Group,
  TextInput,
  Textarea,
  Skeleton,
  Alert,
  ScrollArea,
  rem,
  Text,
  Badge,
  Select,
} from "@mantine/core";
import {
  IconAlertTriangle,
  IconCopy,
  IconDeviceGamepad2,
  IconHeartFilled,
} from "@tabler/icons-react";
import { useMediaQuery } from "@mantine/hooks";
import { useModals } from "@mantine/modals";
import { IconAlertCircle } from "@tabler/icons-react";
import { useTranslation } from "react-i18next";
import { useAdmin } from "@/common/hooks/useAdmin";
import { useState, useEffect, useRef, useCallback } from "react";
import { ActionButton, CancelButton } from "@components/buttons";
import { HelpLink } from "@components/HelpLink";
import { HELP_LINKS } from "@/config/helpLinks";
import { useGame, useGameSessionMap, useGameTemplate, useGenres, useThemes, useUpdateGame } from "@/api/hooks";
import type { CreateGameReq, GameTemplate, GameView } from "@/api/types";
import { Label } from "@components/typography";
import { StatusFieldsEditor } from "./StatusFieldsEditor";
import { SharingSection } from "./GameSharingSection";
import { GameCoverSection } from "./GameCoverSection";
import { useDrawCoverOnSave } from "../hooks";
import { useGameNavigation } from "../hooks/useGameNavigation";
import { GamePlayButtons } from "./GamePlayButtons";
import type { CreateGameFormData } from "../types";
import { gameChanges, gameCreate, type GameFormFields } from "../lib/gameUpdate";

interface FormValues {
  name: string;
  description: string;
  isPublic: boolean;
  systemMessageScenario: string;
  systemMessageGameStart: string;
  imageStyle: string;
  statusFields: string;
  genre: string;
  theme: string;
}

/** The form values as the fields of a game update. */
function formFields(v: FormValues): GameFormFields {
  return {
    name: v.name,
    description: v.description,
    public: v.isPublic,
    systemMessageScenario: v.systemMessageScenario,
    systemMessageGameStart: v.systemMessageGameStart,
    imageStyle: v.imageStyle,
    statusFields: v.statusFields,
    genre: v.genre,
    theme: v.theme,
  };
}

/** The template values as the form holds them. */
function templateFormValues(t: GameTemplate): FormValues {
  return {
    name: t.name,
    description: t.description,
    isPublic: t.public,
    systemMessageScenario: t.systemMessageScenario ?? "",
    systemMessageGameStart: t.systemMessageGameStart ?? "",
    imageStyle: t.imageStyle ?? "",
    statusFields: t.statusFields ?? "",
    genre: t.genre || DEFAULT_GENRE,
    theme: t.theme ?? "",
  };
}

/** Games without a genre are adventures. */
const DEFAULT_GENRE = "adventure";

/** The sections of a game the caller may change; the server decides. */
export interface GameEditRights {
  card: boolean;
  content: boolean;
  settings: boolean;
}

interface GameEditModalProps {
  gameId?: string | null;
  opened: boolean;
  onClose: () => void;
  /**
   * Creates the game: the template values with the form changes. Answers
   * with the game as created, so the editor can have its cover drawn.
   */
  onCreate?: (request: CreateGameReq) => Promise<GameView | undefined> | void;
  createLoading?: boolean;
  /** Callback when user clicks Copy button in view mode */
  onCopy?: () => void;
  /** Loading state for copy operation */
  copyLoading?: boolean;
  /** Pre-populate form fields in create mode (e.g. from YAML import or game copy) */
  initialData?: Partial<CreateGameFormData> | null;
  /** Callback when user clicks Sponsor button */
  onSponsor?: () => void;
  /** Whether to show the inline private share section */
  showShareSection?: boolean;
  /** Workshop ID for workshop share mode (auto-uses workshop key) */
  workshopId?: string;
  /** Whether the current user owns this game (used to show admin action warning) */
  isOwner?: boolean;
}

export function GameEditModal({
  gameId,
  opened,
  onClose,
  onCreate,
  createLoading = false,
  onCopy,
  copyLoading = false,
  initialData,
  onSponsor,
  showShareSection = false,
  workshopId,
  isOwner = true,
}: GameEditModalProps) {
  const { t } = useTranslation("common");
  const isMobile = useMediaQuery("(max-width: 48em)");
  const modals = useModals();
  const { isAdmin } = useAdmin();
  const isAdminAction = isAdmin && !isOwner;

  const isCreateMode = !gameId;
  const { data: game, isLoading: gameLoading, error: gameError } = useGame(gameId ?? "");
  // A new game starts from the server's template.
  const {
    data: template,
    isLoading: templateLoading,
    error: templateError,
  } = useGameTemplate(isCreateMode && opened);
  const isLoading = isCreateMode ? templateLoading : gameLoading;
  const error = isCreateMode ? templateError : gameError;
  // What the caller may change comes with the game or the template (may*
  // flags). A new game has no settings yet.
  const rights: GameEditRights = isCreateMode
    ? {
        card: !!template?.mayEditCard,
        content: !!template?.mayEditContent,
        settings: false,
      }
    : {
        card: !!game?.mayEditCard,
        content: !!game?.mayEditContent,
        settings: !!game?.mayEditSettings,
      };
  /** Nothing to change: the dialog only shows the game. */
  const readOnly = !rights.card && !rights.content && !rights.settings;
  const updateGame = useUpdateGame();
  const drawCoverOnSave = useDrawCoverOnSave();
  const { playGame, continueGame, restartGame } = useGameNavigation();
  const { sessionMap } = useGameSessionMap();
  const session = gameId ? sessionMap.get(gameId) : undefined;

  // Form fields
  const [name, setName] = useState("");
  const [description, setDescription] = useState("");
  const [isPublic, setIsPublic] = useState(false);
  const [systemMessageScenario, setSystemMessageScenario] = useState("");
  const [systemMessageGameStart, setSystemMessageGameStart] = useState("");
  const [imageStyle, setImageStyle] = useState("");
  const [statusFields, setStatusFields] = useState("");
  const [genre, setGenre] = useState(DEFAULT_GENRE);
  const [theme, setTheme] = useState("");

  // Validation
  const [nameError, setNameError] = useState("");

  // Track initial values for dirty checking
  const initialValues = useRef<FormValues | null>(null);
  // The template values: a new game sends what differs from them.
  const templateValues = useRef<FormValues | null>(null);

  // Check if form has unsaved changes
  const isDirty = useCallback(() => {
    if (!initialValues.current) return false;
    return (
      name !== initialValues.current.name ||
      description !== initialValues.current.description ||
      isPublic !== initialValues.current.isPublic ||
      systemMessageScenario !== initialValues.current.systemMessageScenario ||
      systemMessageGameStart !== initialValues.current.systemMessageGameStart ||
      imageStyle !== initialValues.current.imageStyle ||
      statusFields !== initialValues.current.statusFields ||
      genre !== initialValues.current.genre ||
      theme !== initialValues.current.theme
    );
  }, [
    name,
    description,
    isPublic,
    systemMessageScenario,
    systemMessageGameStart,
    imageStyle,
    statusFields,
    genre,
    theme,
  ]);

  // Track if we've initialized form values
  const hasInitialized = useRef(false);

  // Initialize form values when game loads (only once)
  /* eslint-disable react-hooks/set-state-in-effect -- Intentional: initialize form from game data */
  useEffect(() => {
    if (!isCreateMode && game && !isLoading && !hasInitialized.current) {
      hasInitialized.current = true;
      const values: FormValues = {
        name: game.name ?? "",
        description: game.description ?? "",
        isPublic: game.public ?? false,
        systemMessageScenario: game.systemMessageScenario ?? "",
        systemMessageGameStart: game.systemMessageGameStart ?? "",
        imageStyle: game.imageStyle ?? "",
        statusFields: game.statusFields ?? "",
        genre: game.genre || DEFAULT_GENRE,
        theme: game.theme ?? "",
      };
      initialValues.current = values;
      setName(values.name);
      setDescription(values.description);
      setIsPublic(values.isPublic);
      setSystemMessageScenario(values.systemMessageScenario);
      setSystemMessageGameStart(values.systemMessageGameStart);
      setImageStyle(values.imageStyle);
      setStatusFields(values.statusFields);
      setGenre(values.genre);
      setTheme(values.theme);
    }
    // Initialize for create mode (optionally with pre-populated data)
    if (isCreateMode && opened && template && !hasInitialized.current) {
      hasInitialized.current = true;
      const base = templateFormValues(template);
      templateValues.current = base;
      const values: FormValues = {
        name: initialData?.name ?? base.name,
        description: initialData?.description ?? base.description,
        isPublic: initialData?.isPublic ?? base.isPublic,
        systemMessageScenario: initialData?.systemMessageScenario ?? base.systemMessageScenario,
        systemMessageGameStart: initialData?.systemMessageGameStart ?? base.systemMessageGameStart,
        imageStyle: initialData?.imageStyle ?? base.imageStyle,
        statusFields: initialData?.statusFields ?? base.statusFields,
        genre: initialData?.genre || base.genre,
        theme: initialData?.theme ?? base.theme,
      };
      initialValues.current = values;
      setName(values.name);
      setDescription(values.description);
      setIsPublic(values.isPublic);
      setSystemMessageScenario(values.systemMessageScenario);
      setSystemMessageGameStart(values.systemMessageGameStart);
      setImageStyle(values.imageStyle);
      setStatusFields(values.statusFields);
      setGenre(values.genre);
      setTheme(values.theme);
    }
    // Reset when modal closes
    if (!opened) {
      hasInitialized.current = false;
      initialValues.current = null;
      templateValues.current = null;
      if (isCreateMode) {
        // Reset form for create mode
        setName("");
        setDescription("");
        setIsPublic(false);
        setSystemMessageScenario("");
        setSystemMessageGameStart("");
        setImageStyle("");
        setStatusFields("");
        setGenre(DEFAULT_GENRE);
        setTheme("");
        setNameError("");
      }
    }
  }, [isCreateMode, game, isLoading, opened, initialData, template]);

  const handleSave = async () => {
    if (!name.trim()) {
      setNameError(t("games.errors.nameRequired"));
      return;
    }
    if (name.trim().length > 70) {
      setNameError(t("games.errors.nameTooLong"));
      return;
    }

    const edited: GameFormFields = {
      name: name.trim(),
      description: description.trim(),
      public: isPublic,
      systemMessageScenario: systemMessageScenario.trim(),
      systemMessageGameStart: systemMessageGameStart.trim(),
      imageStyle: imageStyle.trim(),
      statusFields: statusFields.trim(),
      genre,
      theme,
    };

    if (isCreateMode) {
      if (!templateValues.current) return;
      // A new game: what differs from the template.
      drawCoverOnSave(await onCreate?.(gameCreate(formFields(templateValues.current), edited)) ?? undefined);
    } else if (game?.id && initialValues.current) {
      // Edit mode: send only what changed, so an update never touches a
      // section the caller didn't edit.
      const changes = gameChanges(formFields(initialValues.current), edited);
      if (Object.keys(changes).length === 0) {
        drawCoverOnSave(game);
        onClose();
        return;
      }
      try {
        drawCoverOnSave(await updateGame.mutateAsync({ id: game.id, changes }));
        onClose();
      } catch {
        // Error handled by mutation
      }
    }
  };

  const handleModalClose = () => {
    if (isDirty()) {
      modals.openConfirmModal({
        title: t("games.editModal.unsavedChanges.title"),
        children: (
          <Text size="sm">{t("games.editModal.unsavedChanges.message")}</Text>
        ),
        labels: {
          confirm: t("games.editModal.unsavedChanges.discard"),
          cancel: t("games.editModal.unsavedChanges.keepEditing"),
        },
        confirmProps: { color: "red" },
        onConfirm: onClose,
      });
    } else {
      onClose();
    }
  };

  const isLoaderActive = isLoading;
  const isSaving = isCreateMode ? createLoading : updateGame.isPending;

  const modalContent = () => {
    if (isLoaderActive) {
      return (
        <Stack gap="md">
          <Skeleton height={32} width="60%" />
          <Skeleton height={80} />
          <Skeleton height={32} width="40%" />
        </Stack>
      );
    }

    if (error || (isCreateMode ? !template : !game)) {
      return (
        <Alert icon={<IconAlertCircle size={16} />} color="red">
          {t("games.errors.loadFailed")}
        </Alert>
      );
    }

    return (
      <ScrollArea h={isMobile ? "calc(100vh - 180px)" : "calc(85vh - 140px)"}>
        <Stack gap="lg" pr="md">
          {/* Admin action warning */}
          {isAdminAction && (
            <Alert
              color="red"
              icon={<IconAlertTriangle size={16} />}
              title={t("games.editModal.adminActionTitle")}
            >
              {t("games.editModal.adminActionWarning")}
            </Alert>
          )}

          {/* Read-only notice */}
          {readOnly && (
            <Alert
              icon={<IconAlertCircle size={16} />}
              color="blue"
              variant="light"
            >
              {t("games.viewModal.readOnlyNotice")}
            </Alert>
          )}

          {/* Game Design section */}
          <Group gap="xs" align="center">
            <IconDeviceGamepad2 size={22} />
            <Text size="lg" fw={700}>
              {t("games.editModal.gameDesignSection")}
            </Text>
          </Group>

          {/* Name */}
          <TextInput
            label={t("games.editFields.name")}
            placeholder={t("games.createModal.namePlaceholder")}
            value={name}
            onChange={(e) => {
              setName(e.target.value);
              if (nameError) setNameError("");
            }}
            error={nameError}
            required
            readOnly={!rights.card}
            maxLength={70}
            description={
              !rights.card
                ? undefined
                : isCreateMode
                  ? `${name.length}/70 · ${t("games.privacyHintShort")}`
                  : `${name.length}/70`
            }
            data-autofocus
          />

          <GenreSelect value={genre} onChange={setGenre} readOnly={!rights.card} />

          {/* Scenario */}
          <Textarea
            label={t("games.editFields.scenario")}
            description={t("games.editFields.scenarioHint")}
            placeholder={t("games.editFields.scenarioPlaceholder")}
            value={systemMessageScenario}
            onChange={(e) => setSystemMessageScenario(e.target.value)}
            minRows={6}
            autosize
            maxRows={12}
            required
            readOnly={!rights.content}
          />

          {/* Game Start */}
          <Textarea
            label={t("games.editFields.gameStart")}
            description={t("games.editFields.gameStartHint")}
            placeholder={t("games.editFields.gameStartPlaceholder")}
            value={systemMessageGameStart}
            onChange={(e) => setSystemMessageGameStart(e.target.value)}
            minRows={4}
            autosize
            maxRows={8}
            required
            readOnly={!rights.content}
          />

          {/* Image Style */}
          <TextInput
            label={t("games.editFields.imageStyle")}
            description={t("games.editFields.imageStyleHint")}
            value={imageStyle}
            onChange={(e) => setImageStyle(e.target.value)}
            placeholder="e.g., pixel art, watercolor, realistic..."
            required
            readOnly={!rights.content}
          />

          {/* Cover */}
          <Stack gap="xs">
            <Label>{t("games.cover.title")}</Label>
            <GameCoverSection
              game={isCreateMode ? undefined : game}
              mayEdit={rights.card}
            />
          </Stack>

          {/* Status Fields */}
          <StatusFieldsEditor
            value={statusFields}
            onChange={setStatusFields}
            readOnly={!rights.content}
          />

          <ThemeSelect value={theme} onChange={setTheme} readOnly={!rights.card} />

          {/* Description - moved to end */}
          <Textarea
            label={t("games.editFields.description")}
            placeholder={t("games.createModal.descriptionPlaceholder")}
            description={isCreateMode ? t("games.privacyHintShort") : undefined}
            value={description}
            onChange={(e) => setDescription(e.target.value)}
            minRows={3}
            readOnly={!rights.card}
          />

          {/* Sharing & Visibility section */}
          <SharingSection
            game={game}
            gameId={isCreateMode ? undefined : gameId!}
            isPublic={isPublic}
            setIsPublic={setIsPublic}
            mayEditCard={rights.card}
            mayPublish={!!(isCreateMode ? template : game)?.mayPublish}
            wasPublic={!!(isCreateMode ? template : game)?.public}
            mayShare={!!game?.mayShare}
            onSponsor={!isCreateMode ? onSponsor : undefined}
            showShareSection={!isCreateMode && showShareSection}
            workshopId={workshopId}
          />
        </Stack>
      </ScrollArea>
    );
  };

  return (
    <Modal
      opened={opened}
      onClose={handleModalClose}
      title={
        <Group gap="sm">
          <Text fw={600}>
            {isCreateMode
              ? t("games.createModal.title")
              : readOnly
                ? t("games.viewModal.title")
                : t("games.editModal.title")}
          </Text>
          <HelpLink href={HELP_LINKS.GAME_TIPS} />
          {!isCreateMode && game?.publicSponsoredApiKeyShareId && (
            <Badge
              size="sm"
              color="pink"
              variant="light"
              leftSection={<IconHeartFilled size={12} />}
            >
              {t("games.sponsor.sponsored")}
            </Badge>
          )}
        </Group>
      }
      size={isMobile ? "100%" : rem(1000)}
      fullScreen={isMobile}
      centered={!isMobile}
      styles={{
        content: {
          maxHeight: isMobile ? undefined : "85vh",
          ...(isAdminAction ? { outline: "3px solid var(--mantine-color-red-6)" } : {}),
        },
        body: { maxHeight: isMobile ? undefined : "calc(85vh - 60px)" },
      }}
    >
      <Stack gap="md">
        {modalContent()}

        <Group justify="space-between" align="flex-start" mt="md" gap="sm">
          {/* The game page plays the game too, or says why it cannot. */}
          {!isCreateMode && game ? (
            <div style={{ width: 180 }}>
              <GamePlayButtons
                game={game}
                hasSession={!!session}
                session={session}
                onPlay={playGame}
                onContinue={continueGame}
                onRestart={restartGame}
                labels={{
                  play: t("allGames.play"),
                  continue: t("allGames.continue"),
                  restart: t("allGames.restart"),
                }}
              />
            </div>
          ) : (
            <span />
          )}
          <Group justify="flex-end" gap="sm">
            {readOnly ? (
              <>
                <CancelButton onClick={onClose}>{t("close")}</CancelButton>
                {onCopy && (
                  <ActionButton
                    onClick={onCopy}
                    loading={copyLoading}
                    size="sm"
                    leftSection={<IconCopy size={16} />}
                  >
                    {t("copyGame")}
                  </ActionButton>
                )}
              </>
            ) : (
              <>
                <CancelButton onClick={handleModalClose} disabled={isSaving}>
                  {t("cancel")}
                </CancelButton>
                <ActionButton onClick={handleSave} loading={isSaving} size="sm">
                  {isCreateMode ? t("games.createModal.submit") : t("save")}
                </ActionButton>
              </>
            )}
          </Group>
        </Group>
      </Stack>
    </Modal>
  );
}

interface ChoiceProps {
  value: string;
  onChange: (value: string) => void;
  readOnly: boolean;
}

/** The engine's genres, each with the features a key must offer to play it. */
function GenreSelect({ value, onChange, readOnly }: ChoiceProps) {
  const { t } = useTranslation("common");
  const { data: genres } = useGenres();

  const needsOf = (id: string) => {
    const needs = genres?.find((g) => g.id === id)?.needs ?? [];
    if (needs.length === 0) return "";
    const features = needs
      .map((toggle) => t(`games.features.${toggle}`, { defaultValue: toggle }))
      .join(", ");
    return t("games.editFields.genreNeeds", { features });
  };
  const options: { value: string; label: string }[] = (genres ?? []).map((g) => ({
    value: g.id ?? "",
    label: t(`games.genres.${g.id}`, { defaultValue: g.id }),
  }));
  if (!options.some((o) => o.value === value)) {
    options.push({ value, label: t(`games.genres.${value}`, { defaultValue: value }) });
  }
  const selectedNeeds = needsOf(value);

  return (
    <Select
      label={t("games.editFields.genre")}
      description={selectedNeeds || t("games.editFields.genreHint")}
      data={options}
      value={value}
      onChange={(next) => next && onChange(next)}
      allowDeselect={false}
      readOnly={readOnly}
      renderOption={({ option }) => {
        const needs = needsOf(option.value);
        return (
          <Stack gap={0}>
            <Text size="sm">{option.label}</Text>
            {needs && (
              <Text size="xs" c="dimmed">
                {needs}
              </Text>
            )}
          </Stack>
        );
      }}
    />
  );
}

/** The playback-engine's looks; empty is the default look. */
function ThemeSelect({ value, onChange, readOnly }: ChoiceProps) {
  const { t } = useTranslation("common");
  const { data: themes } = useThemes();

  const label = (id: string) =>
    t(`games.theme.presets.${id}`, { defaultValue: id });
  const options = (themes ?? [])
    .map((theme) => theme.id ?? "")
    .filter((id) => id !== "")
    .map((id) => ({ value: id, label: label(id) }));
  if (value && !options.some((o) => o.value === value)) {
    options.push({ value, label: label(value) });
  }

  return (
    <Select
      label={t("games.editFields.theme")}
      description={t("games.editFields.themeHint")}
      placeholder={t("games.editFields.themeDefault")}
      data={options}
      value={value || null}
      onChange={(next) => onChange(next ?? "")}
      clearable
      searchable
      readOnly={readOnly}
    />
  );
}
