import { useCallback, useState, type ComponentType } from "react";
import type { ObjGameTheme } from "@/api/generated";
import { mapApiThemeToPartial } from "../types";
import type { PartialGameTheme } from "../theme/types";
import type { MessageTextWrapperProps } from "../theme/presets/types";
import { PRESETS } from "../theme";

interface UseGameThemeResolutionOptions {
  sessionId: string | null;
  apiTheme: ObjGameTheme | null;
  useNeutralTheme: boolean;
  setUseNeutralTheme: (value: boolean) => void;
}

export interface GameThemeResolution {
  effectiveTheme: PartialGameTheme | undefined;
  BackgroundComponent: React.ComponentType | undefined;
  GameMessageWrapper: ComponentType<MessageTextWrapperProps> | undefined;
  PlayerMessageWrapper: ComponentType<MessageTextWrapperProps> | undefined;
  StreamingMessageWrapper: ComponentType<MessageTextWrapperProps> | undefined;
  handleThemeChange: (theme: PartialGameTheme) => void;
  handleNeutralThemeToggle: () => void;
}

export function useGameThemeResolution({
  sessionId,
  apiTheme,
  useNeutralTheme,
  setUseNeutralTheme,
}: UseGameThemeResolutionOptions): GameThemeResolution {
  const [themeOverridesBySessionId, setThemeOverridesBySessionId] = useState<
    Record<string, PartialGameTheme>
  >({});

  const themeOverride = sessionId
    ? (themeOverridesBySessionId[sessionId] ?? null)
    : null;

  const handleThemeChange = useCallback(
    (theme: PartialGameTheme) => {
      if (!sessionId) return;
      setThemeOverridesBySessionId((prev) => ({
        ...prev,
        [sessionId]: theme,
      }));
    },
    [sessionId],
  );

  const handleNeutralThemeToggle = useCallback(() => {
    // Clear theme override when toggling neutral theme
    // This ensures clean switch between default preset and original API theme
    if (sessionId) {
      setThemeOverridesBySessionId((prev) => {
        const newOverrides = { ...prev };
        delete newOverrides[sessionId];
        return newOverrides;
      });
    }
    setUseNeutralTheme(!useNeutralTheme);
  }, [sessionId, useNeutralTheme, setUseNeutralTheme]);

  // Resolve theme: test override > neutral default > AI-generated
  const baseTheme = useNeutralTheme
    ? PRESETS.default.theme
    : mapApiThemeToPartial(apiTheme);
  const effectiveTheme = themeOverride ?? baseTheme;

  // Resolve custom BackgroundComponent from preset (if any)
  const presetName = useNeutralTheme ? "default" : apiTheme?.preset;
  const activePreset = presetName ? PRESETS[presetName] : undefined;
  const BackgroundComponent = activePreset?.BackgroundComponent;
  const GameMessageWrapper = activePreset?.GameMessageWrapper;
  const PlayerMessageWrapper = activePreset?.PlayerMessageWrapper;
  const StreamingMessageWrapper = activePreset?.StreamingMessageWrapper;

  return {
    effectiveTheme,
    BackgroundComponent,
    GameMessageWrapper,
    PlayerMessageWrapper,
    StreamingMessageWrapper,
    handleThemeChange,
    handleNeutralThemeToggle,
  };
}
