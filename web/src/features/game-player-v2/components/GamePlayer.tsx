import { Box } from "@mantine/core";
import { GamePlayerProvider } from "../context";
import type { GamePlayerContextValue } from "../context";
import { useSessionLifecycle } from "../hooks/useSessionLifecycle";
import { useGamePlayerSettings } from "../hooks/useGamePlayerSettings";
import { useGameThemeResolution } from "../hooks/useGameThemeResolution";
import { GameThemeProvider, useGameTheme } from "../theme";
import { GamePlayerHeader } from "./GamePlayerHeader";
import { GameStateScreen } from "./GameStateScreen";
import { MessageList } from "./MessageList";
import { StatusBar } from "./StatusBar";
import { ImageLightbox } from "./ImageLightbox";
import { BackgroundAnimation } from "./BackgroundAnimation";
import { WorkshopPausedOverlay } from "@/features/my-workshop/components/WorkshopPausedOverlay";
import classes from "./GamePlayer.module.css";

/** Scene area with theme-aware background animation */
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

interface GamePlayerProps {
  gameId?: string;
  sessionId?: string;
}

export function GamePlayer({ gameId, sessionId }: GamePlayerProps) {
  // ── Hooks ──────────────────────────────────────────────────────────────
  const lifecycle = useSessionLifecycle({ gameId, sessionId });
  const settings = useGamePlayerSettings();
  const themeResolution = useGameThemeResolution({
    sessionId: lifecycle.state.sessionId,
    apiTheme: lifecycle.state.theme,
    useNeutralTheme: settings.useNeutralTheme,
    setUseNeutralTheme: settings.setUseNeutralTheme,
  });

  // ── State screens (loading, errors, etc.) ──────────────────────────────
  const stateScreen = GameStateScreen({
    phase: lifecycle.state.phase,
    isContinuation: lifecycle.isContinuation,
    isInWorkshopContext: lifecycle.isInWorkshopContext,
    gameLoading: lifecycle.gameLoading,
    gameError: lifecycle.gameError,
    gameExists: lifecycle.gameExists,
    missingFields: lifecycle.missingFields,
    isNoApiKeyError: lifecycle.isNoApiKeyError,
    error: lifecycle.state.error,
    errorObject: lifecycle.state.errorObject,
    onBack: lifecycle.handleBack,
  });

  if (stateScreen) return stateScreen;

  const hasAudioOut = lifecycle.state.messages.some((m) => !!m.hasAudioOut);

  // ── Context value ──────────────────────────────────────────────────────
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

  // ── Main game UI ───────────────────────────────────────────────────────
  return (
    <GameThemeProvider
      theme={themeResolution.effectiveTheme}
      BackgroundComponent={themeResolution.BackgroundComponent}
      GameMessageWrapper={themeResolution.GameMessageWrapper}
      PlayerMessageWrapper={themeResolution.PlayerMessageWrapper}
      StreamingMessageWrapper={themeResolution.StreamingMessageWrapper}
    >
      <GamePlayerProvider value={contextValue}>
        <Box className={classes.container} style={{ position: "relative" }}>
          {lifecycle.isPausedForUser && <WorkshopPausedOverlay />}
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
              apiKeyUnavailable={!lifecycle.apiKeyAvailable}
              audioEnabled={lifecycle.state.messages.some((m) => m.hasAudioIn)}
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
