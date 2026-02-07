import { useCallback, useEffect, useRef, useState } from "react";
import {
  Group,
  Text,
  Loader,
  ActionIcon,
  Tooltip,
  Box,
  Stack,
  Menu,
  Checkbox,
} from "@mantine/core";
import { useNavigate, useRouter } from "@tanstack/react-router";
import { useQueryClient } from "@tanstack/react-query";
import {
  queryKeys,
  useGame,
  useWorkshopEvents,
} from "@/api/hooks";
import { useTranslation } from "react-i18next";
import {
  IconArrowLeft,
  IconAlertCircle,
  IconTextIncrease,
  IconTextDecrease,
  IconSettings,
  IconKey,
} from "@tabler/icons-react";
import env from "@/config/env";
import { ActionButton, TextButton } from "@components/buttons";
import { ErrorModal } from "@/common/components/ErrorModal";
import { useResponsiveDesign } from "@/common/hooks/useResponsiveDesign";
import { useAuth } from "@/providers/AuthProvider";
import { useWorkshopMode } from "@/providers/WorkshopModeProvider";
import { extractRawErrorCode } from "@/common/types/errorCodes";
import { useGameSession } from "../hooks/useGameSession";
import { showErrorModal } from "@/common/lib/globalErrorModal";
import { GamePlayerProvider } from "../context";
import type { GamePlayerContextValue, FontSize } from "../context";
import { mapApiThemeToPartial } from "../types";
import type { PartialGameTheme } from "../theme/types";
import { GameThemeProvider, useGameTheme, PRESET_THEMES } from "../theme";
import { ThemeTestPanel } from "./ThemeTestPanel";
import { SceneCard } from "./SceneCard";
import { PlayerAction } from "./PlayerAction";
import { SystemMessage } from "./SystemMessage";
import { SceneDivider } from "./SceneDivider";
import { TypingIndicator } from "./TypingIndicator";
import { StatusBar } from "./StatusBar";
import { PlayerInput } from "./PlayerInput";
import { ImageLightbox } from "./ImageLightbox";
import { BackgroundAnimation } from "./BackgroundAnimation";
import classes from "./GamePlayer.module.css";

const FONT_SIZES: FontSize[] = ["xs", "sm", "md", "lg", "xl", "2xl", "3xl"];

/** Scene area with theme-aware background animation */
interface SceneAreaWithThemeProps {
  renderMessages: () => React.ReactNode[];
  sceneEndRef: React.RefObject<HTMLDivElement | null>;
  animationEnabled: boolean;
}

function SceneAreaWithTheme({
  renderMessages,
  sceneEndRef,
  animationEnabled,
}: SceneAreaWithThemeProps) {
  const { cssVars, theme } = useGameTheme();
  const animation = theme.background.animation || "none";

  return (
    <Box
      className={classes.sceneArea}
      style={{ ...cssVars }}
    >
      <BackgroundAnimation
        animation={animation}
        disabled={!animationEnabled}
      />
      <div className={classes.messagesScroll}>
        <div
          className={classes.scenesContainer}
          style={{ padding: 'var(--mantine-spacing-md)' }}
        >
          {renderMessages()}
          <div ref={sceneEndRef} />
        </div>
      </div>
    </Box>
  );
}

/** Header with theme-aware styling */
interface HeaderWithThemeProps {
  children: React.ReactNode;
}

function HeaderWithTheme({ children }: HeaderWithThemeProps) {
  const { cssVars } = useGameTheme();

  return (
    <Box className={classes.header} px="md" py="sm" style={cssVars}>
      {children}
    </Box>
  );
}

interface GamePlayerProps {
  gameId?: string;
  sessionId?: string;
}

export function GamePlayer({ gameId, sessionId }: GamePlayerProps) {
  const { t } = useTranslation("common");
  const router = useRouter();
  const navigate = useNavigate();
  const queryClient = useQueryClient();
  const sceneEndRef = useRef<HTMLDivElement>(null);
  const isContinuation = !!sessionId;
  const { isMobile } = useResponsiveDesign();
  const { isParticipant, backendUser } = useAuth();
  const { isInWorkshopMode, activeWorkshopId } = useWorkshopMode();

  // User is in workshop context if they are a participant OR staff/head/individual in workshop mode
  const isInWorkshopContext = isParticipant || isInWorkshopMode;

  // Determine workshop ID for SSE subscription
  const workshopIdForEvents = isParticipant
    ? backendUser?.role?.workshop?.id
    : isInWorkshopMode
      ? (activeWorkshopId ?? undefined)
      : undefined;

  // Subscribe to workshop SSE events when in workshop context
  useWorkshopEvents({
    workshopId: workshopIdForEvents,
  });

  const [lightboxImage, setLightboxImage] = useState<{
    url: string;
    alt?: string;
  } | null>(null);
  const [fontSize, setFontSize] = useState<FontSize>("md");
  const [debugMode, setDebugMode] = useState(false);
  const [animationEnabled, setAnimationEnabled] = useState(true);
  const [useNeutralTheme, setUseNeutralTheme] = useState(false);
  const [isImageGenerationDisabled, setIsImageGenerationDisabled] =
    useState(false);

  const {
    data: game,
    isLoading: gameLoading,
    error: gameError,
  } = useGame(isContinuation ? undefined : gameId);
  const {
    state,
    startSession,
    sendAction,
    retryLastAction,
    loadExistingSession,
    updateSessionApiKey,
    clearStreamError,
    resetGame,
  } = useGameSession(gameId || "");

  const [themeOverridesBySessionId, setThemeOverridesBySessionId] = useState<
    Record<string, PartialGameTheme>
  >({});
  const themeOverride = state.sessionId
    ? (themeOverridesBySessionId[state.sessionId] ?? null)
    : null;

  const handleThemeChange = useCallback(
    (theme: PartialGameTheme) => {
      if (!state.sessionId) return;
      setThemeOverridesBySessionId((prev) => ({
        ...prev,
        [state.sessionId as string]: theme,
      }));
    },
    [state.sessionId],
  );

  const handleNeutralThemeToggle = useCallback(() => {
    // Clear theme override when toggling neutral theme
    // This ensures clean switch between default preset and original API theme
    if (state.sessionId) {
      setThemeOverridesBySessionId((prev) => {
        const newOverrides = { ...prev };
        delete newOverrides[state.sessionId as string];
        return newOverrides;
      });
    }
    setUseNeutralTheme((prev) => !prev);
  }, [state.sessionId]);

  // Load existing session (continuation)
  useEffect(() => {
    if (sessionId && state.phase === "idle") {
      loadExistingSession(sessionId);
    }
  }, [sessionId, state.phase, loadExistingSession]);

  // Auto-start new sessions: API key is resolved server-side
  const autoStartAttemptedRef = useRef(false);
  useEffect(() => {
    if (isContinuation || state.phase !== "idle" || autoStartAttemptedRef.current) return;
    if (gameLoading || gameError || !game) return;
    autoStartAttemptedRef.current = true;
    startSession();
  }, [isContinuation, state.phase, gameLoading, gameError, game, startSession]);

  // Auto-resolve API key for sessions that lost their key (needs-api-key phase)
  useEffect(() => {
    if (state.phase === "needs-api-key") {
      updateSessionApiKey();
    }
  }, [state.phase, updateSessionApiKey]);

  // Debug: Log received theme
  useEffect(() => {
    if (state.theme) {
      console.log(
        "[GamePlayer] Received theme from session:",
        JSON.stringify(state.theme, null, 2),
      );
    }
  }, [state.theme]);

  const displayGame = isContinuation ? state.gameInfo : game;

  const scrollToBottom = useCallback(() => {
    setTimeout(() => {
      sceneEndRef.current?.scrollIntoView({ behavior: "smooth" });
    }, 100);
  }, []);

  const prevMessageCountRef = useRef(state.messages.length);
  useEffect(() => {
    // Only auto-scroll when a new message is added, not on streaming text updates
    if (state.messages.length !== prevMessageCountRef.current) {
      prevMessageCountRef.current = state.messages.length;
      scrollToBottom();
    }
  }, [state.messages, scrollToBottom]);

  // Show global error modal for recoverable mid-game errors (AI errors, send failures)
  useEffect(() => {
    if (state.streamError) {
      showErrorModal({
        code: state.streamError.code ?? undefined,
        message: !state.streamError.code
          ? state.streamError.message
          : undefined,
        onDismiss: clearStreamError,
      });
    }
  }, [state.streamError, clearStreamError]);

  const handleSendAction = async (message: string) => {
    await sendAction(message);
  };

  const handleBack = () => {
    // Invalidate queries so the games/sessions lists refresh with any new sessions
    queryClient.invalidateQueries({ queryKey: queryKeys.games });
    queryClient.invalidateQueries({ queryKey: queryKeys.userSessions });
    // Navigate back to wherever the user came from (My Games, All Games, Sessions, etc.)
    if (window.history.length > 1) {
      router.history.back();
    } else {
      // Fallback if there's no history (e.g. direct URL access)
      navigate({ to: "/" });
    }
  };

  // Check if the error is a "no API key" error
  const isNoApiKeyError =
    state.phase === "error" && extractRawErrorCode(state.errorObject) === "no_api_key";

  const openLightbox = useCallback((url: string, alt?: string) => {
    setLightboxImage({ url, alt });
  }, []);

  const closeLightbox = useCallback(() => {
    setLightboxImage(null);
  }, []);

  const increaseFontSize = useCallback(() => {
    setFontSize((current) => {
      const idx = FONT_SIZES.indexOf(current);
      return idx < FONT_SIZES.length - 1 ? FONT_SIZES[idx + 1] : current;
    });
  }, []);

  const decreaseFontSize = useCallback(() => {
    setFontSize((current) => {
      const idx = FONT_SIZES.indexOf(current);
      return idx > 0 ? FONT_SIZES[idx - 1] : current;
    });
  }, []);

  const toggleDebugMode = useCallback(() => {
    setDebugMode((current) => !current);
  }, []);

  const disableImageGeneration = useCallback((errorCode: string) => {
    setIsImageGenerationDisabled(true);
    showErrorModal({ code: errorCode });
  }, []);

  // Use flex: 1 to fill available space between app header and footer
  const containerHeight = undefined;

  const contextValue: GamePlayerContextValue = {
    state,
    startSession,
    sendAction,
    retryLastAction,
    loadExistingSession,
    resetGame,
    openLightbox,
    closeLightbox,
    lightboxImage,
    fontSize,
    increaseFontSize,
    decreaseFontSize,
    debugMode,
    toggleDebugMode,
    isImageGenerationDisabled,
    disableImageGeneration,
  };

  if (gameLoading || (isContinuation && state.phase === "idle")) {
    return (
      <Box className={classes.container} h={containerHeight}>
        <Stack
          className={classes.stateContainer}
          align="center"
          justify="center"
          gap="md"
        >
          <Loader size="lg" color="accent" />
          <Text c="dimmed">{t("gamePlayer.loading.game")}</Text>
        </Stack>
      </Box>
    );
  }

  if (!isContinuation && (gameError || !game)) {
    return (
      <Box className={classes.container} h={containerHeight}>
        <Stack
          className={classes.stateContainer}
          align="center"
          justify="center"
          gap="md"
        >
          <IconAlertCircle size={48} color="var(--mantine-color-red-5)" />
          <Text size="lg" fw={600}>
            {t("gamePlayer.error.gameNotFound")}
          </Text>
          <TextButton
            onClick={handleBack}
            leftSection={<IconArrowLeft size={16} />}
          >
            {t("gamePlayer.error.backToGames")}
          </TextButton>
        </Stack>
      </Box>
    );
  }

  // Validate required game fields before allowing play
  const getMissingFields = () => {
    if (!game || isContinuation) return [];
    const missing: string[] = [];
    if (!game.systemMessageScenario?.trim())
      missing.push(t("games.editFields.scenario"));
    if (!game.systemMessageGameStart?.trim())
      missing.push(t("games.editFields.gameStart"));
    if (!game.imageStyle?.trim())
      missing.push(t("games.editFields.imageStyle"));
    return missing;
  };

  const missingFields = getMissingFields();
  if (!isContinuation && missingFields.length > 0) {
    return (
      <Box className={classes.container} h={containerHeight}>
        <Stack
          className={classes.stateContainer}
          align="center"
          justify="center"
          gap="md"
        >
          <IconAlertCircle size={48} color="var(--mantine-color-red-5)" />
          <Text size="lg" fw={600}>
            {t("gamePlayer.error.missingFields")}
          </Text>
          <Text c="dimmed" ta="center">
            {missingFields.join(", ")}
          </Text>
          <TextButton
            onClick={handleBack}
            leftSection={<IconArrowLeft size={16} />}
          >
            {t("gamePlayer.error.backToGames")}
          </TextButton>
        </Stack>
      </Box>
    );
  }

  // "No API key" error — show specific UX depending on context
  if (isNoApiKeyError) {
    return (
      <Box className={classes.container} h={containerHeight}>
        <Stack
          className={classes.stateContainer}
          align="center"
          justify="center"
          gap="md"
        >
          <IconAlertCircle size={48} color="var(--mantine-color-red-5)" />
          <Text size="lg" fw={600}>
            {t("gamePlayer.error.noApiKey.title", "No API Key Available")}
          </Text>
          <Text c="dimmed" ta="center" maw={400}>
            {isInWorkshopContext
              ? t(
                  "gamePlayer.error.noApiKey.workshop",
                  "No API key is configured for this workshop. Please contact your workshop administrator.",
                )
              : t(
                  "gamePlayer.error.noApiKey.personal",
                  "You need to configure an API key before you can play. Go to your API Key settings to add one.",
                )}
          </Text>
          <Group gap="sm">
            <TextButton
              onClick={handleBack}
              leftSection={<IconArrowLeft size={16} />}
            >
              {t("gamePlayer.error.backToGames")}
            </TextButton>
            {!isInWorkshopContext && (
              <ActionButton
                leftSection={<IconKey size={16} />}
                onClick={() => navigate({ to: "/api-keys" })}
              >
                {t("gamePlayer.error.noApiKey.goToSettings", "API Key Settings")}
              </ActionButton>
            )}
          </Group>
        </Stack>
      </Box>
    );
  }

  // Generic error state
  if (state.phase === "error") {
    return (
      <>
        <Box className={classes.container} h={containerHeight}>
          <Stack
            className={classes.stateContainer}
            align="center"
            justify="center"
            gap="md"
          >
            <Loader size="lg" color="accent" />
          </Stack>
        </Box>
        <ErrorModal
          opened={true}
          onClose={handleBack}
          error={state.errorObject}
          message={!state.errorObject ? state.error || undefined : undefined}
          title={t("gamePlayer.error.sessionFailed")}
        />
      </>
    );
  }

  // Session exists but API key was deleted — auto-resolving server-side
  if (state.phase === "needs-api-key") {
    return (
      <Box className={classes.container} h={containerHeight}>
        <Stack
          className={classes.stateContainer}
          align="center"
          justify="center"
          gap="md"
        >
          <Loader size="lg" color="accent" />
          <Text fw={600}>{t("gamePlayer.loading.resolvingKey", "Resolving API key...")}</Text>
        </Stack>
      </Box>
    );
  }

  if (state.phase === "idle" || state.phase === "starting") {
    return (
      <Box className={classes.container} h={containerHeight}>
        <Stack
          className={classes.stateContainer}
          align="center"
          justify="center"
          gap="md"
        >
          <Loader size="lg" color="accent" />
          <Text fw={600}>{t("gamePlayer.loading.starting")}</Text>
          <Text c="dimmed" size="sm">
            {t("gamePlayer.loading.startingHint")}
          </Text>
        </Stack>
      </Box>
    );
  }

  const showImages = !isImageGenerationDisabled;

  const renderMessages = () => {
    const elements: React.ReactNode[] = [];

    // Track previous game message's status fields for showing changes
    let previousGameStatusFields: (typeof state.messages)[0]["statusFields"] =
      undefined;

    state.messages.forEach((message, index) => {
      if (message.type === "player") {
        elements.push(
          <PlayerAction
            key={message.id}
            text={message.text}
            error={message.error}
            errorCode={message.errorCode}
            onRetry={message.error ? retryLastAction : undefined}
          />,
        );
      } else if (message.type === "system") {
        elements.push(<SystemMessage key={message.id} message={message} />);
      } else {
        if (index > 0 && state.messages[index - 1]?.type !== "system") {
          elements.push(<SceneDivider key={`divider-${message.id}`} />);
        }
        elements.push(
          <SceneCard
            key={message.id}
            message={message}
            showImages={showImages}
            previousStatusFields={previousGameStatusFields}
          />,
        );
        // Update previous status fields for next game message
        if (message.statusFields?.length) {
          previousGameStatusFields = message.statusFields;
        }
      }
    });

    if (state.isWaitingForResponse && state.messages.length > 0) {
      const lastMessage = state.messages[state.messages.length - 1];
      if (lastMessage.type === "player" || !lastMessage.isStreaming) {
        elements.push(<TypingIndicator key="typing" />);
      }
    }

    // Show input inline when user can type
    if (!state.isWaitingForResponse && state.messages.length > 0) {
      const inputClasses = classes.inlineInput;
      elements.push(
        <div key="inline-input" className={inputClasses}>
          <PlayerInput
            onSend={handleSendAction}
            disabled={state.isWaitingForResponse}
            placeholder={t("gamePlayer.input.placeholder")}
          />
        </div>,
      );
    }

    return elements;
  };

  // Deep merge API theme with local override for testing
  // If neutral theme is enabled, use the default preset
  const baseTheme = useNeutralTheme
    ? PRESET_THEMES.default
    : mapApiThemeToPartial(state.theme);
  const effectiveTheme = themeOverride
    ? {
        corners: { ...baseTheme?.corners, ...themeOverride.corners },
        background: { ...baseTheme?.background, ...themeOverride.background },
        player: { ...baseTheme?.player, ...themeOverride.player },
        gameMessage: {
          ...baseTheme?.gameMessage,
          ...themeOverride.gameMessage,
        },
        cards: { ...baseTheme?.cards, ...themeOverride.cards },
        thinking: { ...baseTheme?.thinking, ...themeOverride.thinking },
        typography: { ...baseTheme?.typography, ...themeOverride.typography },
        statusFields: {
          ...baseTheme?.statusFields,
          ...themeOverride.statusFields,
        },
        header: { ...baseTheme?.header, ...themeOverride.header },
        divider: { ...baseTheme?.divider, ...themeOverride.divider },
        statusEmojis: {
          ...baseTheme?.statusEmojis,
          ...themeOverride.statusEmojis,
        },
      }
    : baseTheme;

  return (
    <GameThemeProvider theme={effectiveTheme}>
      <GamePlayerProvider value={contextValue}>
        <Box className={classes.container} h={containerHeight}>
          <HeaderWithTheme>
            <Group justify="space-between" wrap="nowrap">
              <Group gap="sm" wrap="nowrap" style={{ minWidth: 0, flex: 1 }}>
                <Tooltip label={t("gamePlayer.header.back")} position="bottom">
                  <ActionIcon
                    variant="subtle"
                    color="gray"
                    onClick={handleBack}
                    aria-label={t("gamePlayer.header.back")}
                    size="lg"
                  >
                    <IconArrowLeft size={20} />
                  </ActionIcon>
                </Tooltip>
                <Box style={{ minWidth: 0, flex: 1 }}>
                  <Text fw={600} truncate size="sm">
                    {displayGame?.name || t("gamePlayer.unnamed")}
                  </Text>
                  {displayGame?.description && (
                    <Text
                      size="xs"
                      truncate
                      className={classes.headerDescription}
                    >
                      {displayGame.description}
                    </Text>
                  )}
                </Box>
              </Group>
              <Group gap="xs" wrap="nowrap">
                <Tooltip
                  label={t("gamePlayer.header.decreaseFont")}
                  position="bottom"
                >
                  <ActionIcon
                    variant="subtle"
                    color="gray"
                    onClick={decreaseFontSize}
                    disabled={fontSize === "xs"}
                    aria-label={t("gamePlayer.header.decreaseFont")}
                    size="lg"
                  >
                    <IconTextDecrease size={18} />
                  </ActionIcon>
                </Tooltip>
                <Tooltip
                  label={t("gamePlayer.header.increaseFont")}
                  position="bottom"
                >
                  <ActionIcon
                    variant="subtle"
                    color="gray"
                    onClick={increaseFontSize}
                    disabled={fontSize === "3xl"}
                    aria-label={t("gamePlayer.header.increaseFont")}
                    size="lg"
                  >
                    <IconTextIncrease size={18} />
                  </ActionIcon>
                </Tooltip>
                <Menu shadow="md" width={200} position="bottom-end">
                  <Menu.Target>
                    <Tooltip
                      label={t("gamePlayer.header.settings")}
                      position="bottom"
                    >
                      <ActionIcon
                        variant="subtle"
                        color="gray"
                        aria-label={t("gamePlayer.header.settings")}
                        size="lg"
                      >
                        <IconSettings size={18} />
                      </ActionIcon>
                    </Tooltip>
                  </Menu.Target>
                  <Menu.Dropdown>
                    <Menu.Label>{t("gamePlayer.header.settings")}</Menu.Label>
                    <Menu.Item
                      closeMenuOnClick={false}
                      onClick={() => setAnimationEnabled(!animationEnabled)}
                    >
                      <Checkbox
                        label={t("gamePlayer.header.disableAnimations")}
                        checked={!animationEnabled}
                        onChange={() => setAnimationEnabled(!animationEnabled)}
                        size="sm"
                        styles={{
                          input: { cursor: "pointer" },
                          label: { cursor: "pointer" },
                        }}
                      />
                    </Menu.Item>
                    <Menu.Item
                      closeMenuOnClick={false}
                      onClick={handleNeutralThemeToggle}
                    >
                      <Checkbox
                        label={t("gamePlayer.header.useNeutralTheme")}
                        checked={useNeutralTheme}
                        onChange={handleNeutralThemeToggle}
                        size="sm"
                        styles={{
                          input: { cursor: "pointer" },
                          label: { cursor: "pointer" },
                        }}
                      />
                    </Menu.Item>
                    <Menu.Divider />
                    <Menu.Item
                      closeMenuOnClick={false}
                      onClick={toggleDebugMode}
                    >
                      <Checkbox
                        label={t("gamePlayer.header.debug")}
                        checked={debugMode}
                        onChange={toggleDebugMode}
                        size="sm"
                        styles={{
                          input: { cursor: "pointer" },
                          label: { cursor: "pointer" },
                        }}
                      />
                    </Menu.Item>
                  </Menu.Dropdown>
                </Menu>
                {env.DEV && (
                  <ThemeTestPanel
                    currentTheme={effectiveTheme}
                    onThemeChange={handleThemeChange}
                  />
                )}
              </Group>
            </Group>
          </HeaderWithTheme>

          <StatusBar statusFields={state.statusFields} />

          <SceneAreaWithTheme
            renderMessages={renderMessages}
            sceneEndRef={sceneEndRef}
            animationEnabled={animationEnabled && !isMobile}
          />

          <ImageLightbox />
        </Box>
      </GamePlayerProvider>
    </GameThemeProvider>
  );
}
