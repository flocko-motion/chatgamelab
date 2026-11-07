import { Box } from "@mantine/core";
import { GamePlayerProvider } from "../context";
import type { GamePlayerContextValue } from "../context";
import type { GuestCopyInfo, GuestStartMode } from "./GuestWelcome";
import { useGuestSessionLifecycle } from "../hooks/useGuestSessionLifecycle";
import { useGamePlayerSettings } from "../hooks/useGamePlayerSettings";
import { useGameThemeResolution } from "../hooks/useGameThemeResolution";
import { GameThemeProvider, useGameTheme } from "../theme";
import { GamePlayerHeader } from "./GamePlayerHeader";
import { GameStateScreen } from "./GameStateScreen";
import { MessageList } from "./MessageList";
import { StatusBar } from "./StatusBar";
import { ImageLightbox } from "./ImageLightbox";
import { BackgroundAnimation } from "./BackgroundAnimation";
import { FONT_SIZE_MAP } from "./SceneCard";
import classes from "./GamePlayer.module.css";
import { modals } from "@mantine/modals";
import { useTranslation } from "react-i18next";
import { useAuth } from "@/providers/AuthProvider";
import { useCopyGameIntent } from "@/common/hooks/useCopyGameIntent";
import { GameEditModal } from "@/features/games/components/GameEditModal";

/** Scene area with theme-aware background animation (shared with GamePlayer) */
interface SceneAreaProps {
  children: React.ReactNode;
  sceneEndRef: React.RefObject<HTMLDivElement | null>;
  animationEnabled: boolean;
}

function SceneArea({
  children,
  sceneEndRef,
  animationEnabled,
}: SceneAreaProps) {
  const { cssVars, theme, BackgroundComponent: CustomBg } = useGameTheme();
  const animation = theme.background.animation || "none";

  return (
    <Box className={classes.sceneArea} style={{ ...cssVars }}>
      {CustomBg && animationEnabled ? (
        <CustomBg />
      ) : (
        <BackgroundAnimation
          animation={animation}
          disabled={!animationEnabled}
        />
      )}
      <div className={classes.messagesScroll}>
        <div
          className={classes.scenesContainer}
          style={{ padding: "var(--mantine-spacing-md)" }}
        >
          {children}
          <div ref={sceneEndRef} />
        </div>
      </div>
    </Box>
  );
}

interface GuestGamePlayerProps {
  token: string;
  mode?: GuestStartMode;
  /** Set for a public game: lets the player copy it while playing. */
  copyInfo?: GuestCopyInfo | null;
  onBack?: () => void;
}

/**
 * Guest game player — anonymous play via private share token.
 * Renders the same visual UI as GamePlayer but uses guest-specific hooks
 * that don't require authentication.
 */
export function GuestGamePlayer({
  token,
  mode = "new",
  copyInfo,
  onBack,
}: GuestGamePlayerProps) {
  const { t } = useTranslation("common");
  const { isAuthenticated } = useAuth();
  const copy = useCopyGameIntent(`/play/${token}`);
  const lifecycle = useGuestSessionLifecycle(token, mode, onBack);

  // Copying while signed in opens a dialogue right here. Without an account it
  // means leaving for the login, which abandons the round — so ask first.
  const startCopy = (gameId: string) => {
    if (isAuthenticated) {
      void copy.start(gameId);
      return;
    }
    modals.openConfirmModal({
      title: t("guestPlay.copyLeaveTitle"),
      children: t("guestPlay.copyLeaveMessage"),
      labels: { confirm: t("guestPlay.copyGameLogin"), cancel: t("cancel") },
      onConfirm: () => void copy.start(gameId),
    });
  };
  const settings = useGamePlayerSettings();
  const themeResolution = useGameThemeResolution({
    sessionId: lifecycle.state.sessionId,
    apiTheme: lifecycle.state.theme,
    useNeutralTheme: settings.useNeutralTheme,
    setUseNeutralTheme: settings.setUseNeutralTheme,
  });

  // State screens (loading, errors, etc.)
  const stateScreen = GameStateScreen({
    phase: lifecycle.state.phase,
    isContinuation: lifecycle.isContinuation,
    isInWorkshopContext: false,
    gameLoading: false,
    gameError: null,
    gameExists: true,
    missingFields: [],
    isNoApiKeyError: false,
    error: lifecycle.state.error,
    errorObject: lifecycle.state.errorObject,
    onBack: lifecycle.handleBack,
    startingProgress: lifecycle.startingProgress,
  });

  if (stateScreen) return stateScreen;

  const hasAudioOut = lifecycle.state.messages.some((m) => !!m.hasAudioOut);

  const contextValue: GamePlayerContextValue = {
    state: lifecycle.state,
    startSession: lifecycle.startSession,
    sendAction: lifecycle.sendAction,
    retryLastAction: lifecycle.retryLastAction,
    loadExistingSession: lifecycle.loadExistingSession,
    resetGame: lifecycle.resetGame,
    openLightbox: settings.openLightbox,
    closeLightbox: settings.closeLightbox,
    lightboxImage: settings.lightboxImage,
    fontSize: settings.fontSize,
    increaseFontSize: settings.increaseFontSize,
    decreaseFontSize: settings.decreaseFontSize,
    resetFontSize: settings.resetFontSize,
    debugMode: settings.debugMode,
    toggleDebugMode: settings.toggleDebugMode,
    textEffectsEnabled: settings.textEffectsEnabled,
    isImageGenerationDisabled: settings.isImageGenerationDisabled,
    disableImageGeneration: settings.disableImageGeneration,
  };

  return (
    <GameThemeProvider
      theme={themeResolution.effectiveTheme}
      BackgroundComponent={themeResolution.BackgroundComponent}
      GameMessageWrapper={themeResolution.GameMessageWrapper}
      PlayerMessageWrapper={themeResolution.PlayerMessageWrapper}
      StreamingMessageWrapper={themeResolution.StreamingMessageWrapper}
    >
      <GamePlayerProvider value={contextValue}>
        <Box className={classes.container} style={{ '--game-font-size': FONT_SIZE_MAP[settings.fontSize] } as React.CSSProperties}>
          <GamePlayerHeader
            gameName={lifecycle.displayGame?.name}
            gameDescription={lifecycle.displayGame?.description}
            sessionLanguage={lifecycle.state.sessionLanguage}
            sessionId={lifecycle.state.sessionId}
            gameId={lifecycle.displayGame?.id}
            messageCount={lifecycle.state.messages.length}
            aiModel={lifecycle.state.aiModel}
            aiPlatform={lifecycle.state.aiPlatform}
            promptConstraintSource={lifecycle.state.promptConstraintSource}
            hasAudioOut={hasAudioOut}
            isAudioMuted={settings.isAudioMuted}
            onToggleAudioMuted={settings.toggleAudioMuted}
            fontSize={settings.fontSize}
            increaseFontSize={settings.increaseFontSize}
            decreaseFontSize={settings.decreaseFontSize}
            resetFontSize={settings.resetFontSize}
            animationEnabled={settings.animationEnabled}
            onToggleAnimation={() =>
              settings.setAnimationEnabled(!settings.animationEnabled)
            }
            textEffectsEnabled={settings.textEffectsEnabled}
            onToggleTextEffects={() =>
              settings.setTextEffectsEnabled(!settings.textEffectsEnabled)
            }
            useNeutralTheme={settings.useNeutralTheme}
            onToggleNeutralTheme={themeResolution.handleNeutralThemeToggle}
            onBack={lifecycle.handleBack}
            onCopyGame={
              copyInfo?.public
                ? () => startCopy(copyInfo.gameId)
                : undefined
            }
            currentTheme={themeResolution.effectiveTheme}
            onThemeChange={themeResolution.handleThemeChange}
          />

          <StatusBar statusFields={lifecycle.state.statusFields} />

          <SceneArea
            sceneEndRef={lifecycle.sceneEndRef}
            animationEnabled={settings.animationEnabled}
          >
            <MessageList
              messages={lifecycle.state.messages}
              isWaitingForResponse={lifecycle.state.isWaitingForResponse}
              isImageGenerationDisabled={settings.isImageGenerationDisabled}
              isAudioMuted={settings.isAudioMuted}
              apiKeyUnavailable={false}
              onSendAction={lifecycle.handleSendAction}
              onRetryLastAction={lifecycle.retryLastAction}
            />
          </SceneArea>

          <ImageLightbox />

          <GameEditModal
            opened={copy.modal.opened}
            onClose={copy.modal.close}
            onCreate={copy.modal.onCreate}
            createLoading={copy.modal.createLoading}
            initialData={copy.modal.initialData}
          />
        </Box>
      </GamePlayerProvider>
    </GameThemeProvider>
  );
}
