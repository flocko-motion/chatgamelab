import { Box } from "@mantine/core";
import { GamePlayerProvider } from "../context";
import type { GamePlayerContextValue } from "../context";
import type { GuestStartMode } from "./GuestWelcome";
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
import classes from "./GamePlayer.module.css";

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
  onBack,
}: GuestGamePlayerProps) {
  const lifecycle = useGuestSessionLifecycle(token, mode, onBack);
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
        <Box className={classes.container}>
          <GamePlayerHeader
            gameName={lifecycle.displayGame?.name}
            gameDescription={lifecycle.displayGame?.description}
            sessionLanguage={lifecycle.state.sessionLanguage}
            aiModel={lifecycle.state.aiModel}
            aiPlatform={lifecycle.state.aiPlatform}
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
        </Box>
      </GamePlayerProvider>
    </GameThemeProvider>
  );
}
