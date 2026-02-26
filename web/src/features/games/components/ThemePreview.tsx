/**
 * ThemePreview - Full game player preview for theme selection.
 *
 * Renders the real game player inner components (StatusBar, MessageList,
 * BackgroundAnimation) with mock data inside both GameThemeProvider and
 * GamePlayerProvider. This gives us real text effects, real dividers,
 * real typing indicator — everything the player would show.
 */

import { useRef, useMemo } from "react";
import { useTranslation } from "react-i18next";
import { Box } from "@mantine/core";
import {
  GameThemeProvider,
  useGameTheme,
  PRESETS,
  type PartialGameTheme,
  type BackgroundAnimation as BackgroundAnimationType,
} from "@/features/game-player-v2/theme";
import {
  GamePlayerProvider,
  type GamePlayerContextValue,
} from "@/features/game-player-v2/context";
import type {
  GamePlayerState,
  SceneMessage,
} from "@/features/game-player-v2/types";
import { StatusBar } from "@/features/game-player-v2/components/StatusBar";
import { MessageList } from "@/features/game-player-v2/components/MessageList";
import { BackgroundAnimation } from "@/features/game-player-v2/components/BackgroundAnimation";
import classes from "@/features/game-player-v2/components/GamePlayer.module.css";
import type { ObjGameTheme, ObjStatusField } from "@/api/generated";

interface ThemePreviewProps {
  /** The theme config to preview (preset + overrides) */
  apiTheme: ObjGameTheme | null;
}

/** Resolve ObjGameTheme → PartialGameTheme + preset wrappers */
function resolveTheme(apiTheme: ObjGameTheme | null) {
  const presetKey =
    apiTheme?.preset && PRESETS[apiTheme.preset] ? apiTheme.preset : "default";
  const presetDef = PRESETS[presetKey];
  const partial: PartialGameTheme = JSON.parse(JSON.stringify(presetDef.theme));

  if (apiTheme?.animation) {
    partial.background = {
      ...partial.background,
      animation: apiTheme.animation as BackgroundAnimationType,
    };
  }

  if (apiTheme?.thinkingText) {
    partial.thinking = {
      ...partial.thinking,
      text: apiTheme.thinkingText,
    };
  }

  return {
    partial,
    BackgroundComponent: presetDef.BackgroundComponent,
    GameMessageWrapper: presetDef.GameMessageWrapper,
    PlayerMessageWrapper: presetDef.PlayerMessageWrapper,
    StreamingMessageWrapper: presetDef.StreamingMessageWrapper,
  };
}

/** Build mock messages for the preview */
function useMockMessages(t: (key: string) => string): SceneMessage[] {
  return useMemo(
    () => [
      {
        id: "preview-game-1",
        type: "game" as const,
        text: t("games.theme.sampleGame"),
        statusFields: [
          { name: "Health", value: "100" },
          { name: "XP", value: "50" },
          { name: "Items", value: "2" },
        ],
        timestamp: new Date(),
      },
      {
        id: "preview-player-1",
        type: "player" as const,
        text: t("games.theme.samplePlayer"),
        timestamp: new Date(),
      },
      {
        id: "preview-game-2",
        type: "game" as const,
        text: t("games.theme.sampleGame2"),
        statusFields: [
          { name: "Health", value: "85" },
          { name: "XP", value: "120" },
          { name: "Items", value: "3" },
        ],
        timestamp: new Date(),
      },
    ],
    [t],
  );
}

/** Build mock status fields (latest values) */
function useMockStatusFields(): ObjStatusField[] {
  return useMemo(
    () => [
      { name: "Health", value: "85" },
      { name: "XP", value: "120" },
      { name: "Items", value: "3" },
    ],
    [],
  );
}

// No-op functions for mock context
const noop = () => { };
const asyncNoop = async () => { };

export function ThemePreview({ apiTheme }: ThemePreviewProps) {
  const { t } = useTranslation("common");
  const sceneEndRef = useRef<HTMLDivElement>(null);
  const messages = useMockMessages(t);
  const statusFields = useMockStatusFields();

  const resolved = useMemo(() => resolveTheme(apiTheme), [apiTheme]);

  // Mock GamePlayerState
  const mockState: GamePlayerState = useMemo(
    () => ({
      phase: "playing",
      sessionId: "preview",
      gameInfo: { id: "preview", name: "Preview" },
      messages,
      statusFields,
      isWaitingForResponse: true,
      error: null,
      errorObject: null,
      streamError: null,
      theme: apiTheme,
      aiModel: null,
      aiPlatform: null,
      sessionLanguage: null,
    }),
    [messages, statusFields, apiTheme],
  );

  // Mock GamePlayerContextValue
  const mockContext: GamePlayerContextValue = useMemo(
    () => ({
      state: mockState,
      startSession: asyncNoop,
      sendAction: asyncNoop,
      loadExistingSession: asyncNoop,
      retryLastAction: noop,
      resetGame: noop,
      openLightbox: noop,
      closeLightbox: noop,
      lightboxImage: null,
      fontSize: "md",
      increaseFontSize: noop,
      decreaseFontSize: noop,
      resetFontSize: noop,
      debugMode: false,
      toggleDebugMode: noop,
      textEffectsEnabled: true,
      isImageGenerationDisabled: true,
      disableImageGeneration: noop,
    }),
    [mockState],
  );

  return (
    <GameThemeProvider
      theme={resolved.partial}
      BackgroundComponent={resolved.BackgroundComponent}
      GameMessageWrapper={resolved.GameMessageWrapper}
      PlayerMessageWrapper={resolved.PlayerMessageWrapper}
      StreamingMessageWrapper={resolved.StreamingMessageWrapper}
    >
      <GamePlayerProvider value={mockContext}>
        <PreviewShell
          statusFields={statusFields}
          messages={messages}
          sceneEndRef={sceneEndRef}
        />
      </GamePlayerProvider>
    </GameThemeProvider>
  );
}

interface PreviewShellProps {
  statusFields: ObjStatusField[];
  messages: SceneMessage[];
  sceneEndRef: React.RefObject<HTMLDivElement | null>;
}

function PreviewShell({
  statusFields,
  messages,
  sceneEndRef,
}: PreviewShellProps) {
  const { cssVars, theme, BackgroundComponent: CustomBg } = useGameTheme();
  const animation = theme.background.animation || "none";

  return (
    <Box className={classes.container} style={{ ...cssVars, height: "100%" }}>
      {/* Status bar with real component */}
      <StatusBar statusFields={statusFields} />

      {/* Scene area — same structure as GamePlayer's SceneArea */}
      <Box className={classes.sceneArea} style={{ ...cssVars }}>
        {CustomBg ? (
          <CustomBg />
        ) : (
          <BackgroundAnimation animation={animation} />
        )}
        <div className={classes.messagesScroll}>
          <div
            className={classes.scenesContainer}
            style={{ padding: "var(--mantine-spacing-md)" }}
          >
            <MessageList
              messages={messages}
              isWaitingForResponse={true}
              isImageGenerationDisabled={true}
              isAudioMuted={false}
              onSendAction={asyncNoop}
              onRetryLastAction={noop}
            />
            <div ref={sceneEndRef} />
          </div>
        </div>
      </Box>
    </Box>
  );
}
