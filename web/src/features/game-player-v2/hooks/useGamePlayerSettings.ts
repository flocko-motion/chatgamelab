import { useCallback, useState } from "react";
import type { FontSize } from "../context";
import { showErrorModal } from "@/common/lib/globalErrorModal";

const FONT_SIZES: FontSize[] = ["xs", "sm", "md", "lg", "xl", "2xl", "3xl"];

export interface GamePlayerSettings {
  // Lightbox
  lightboxImage: { url: string; alt?: string } | null;
  openLightbox: (url: string, alt?: string) => void;
  closeLightbox: () => void;

  // Font size
  fontSize: FontSize;
  increaseFontSize: () => void;
  decreaseFontSize: () => void;
  resetFontSize: () => void;

  // Debug
  debugMode: boolean;
  toggleDebugMode: () => void;

  // Animation
  animationEnabled: boolean;
  setAnimationEnabled: (enabled: boolean) => void;

  // Text effects
  textEffectsEnabled: boolean;
  setTextEffectsEnabled: (enabled: boolean) => void;

  // Neutral theme
  useNeutralTheme: boolean;
  setUseNeutralTheme: (value: boolean) => void;

  // Image generation
  isImageGenerationDisabled: boolean;
  disableImageGeneration: (errorCode: string) => void;
}

export function useGamePlayerSettings(): GamePlayerSettings {
  const [lightboxImage, setLightboxImage] = useState<{
    url: string;
    alt?: string;
  } | null>(null);
  const [fontSize, setFontSize] = useState<FontSize>("md");
  const [debugMode, setDebugMode] = useState(false);
  const [animationEnabled, setAnimationEnabled] = useState(true);
  const [textEffectsEnabled, setTextEffectsEnabled] = useState(true);
  const [useNeutralTheme, setUseNeutralTheme] = useState(false);
  const [isImageGenerationDisabled, setIsImageGenerationDisabled] =
    useState(false);

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

  const resetFontSize = useCallback(() => {
    setFontSize("md");
  }, []);

  const toggleDebugMode = useCallback(() => {
    setDebugMode((current) => !current);
  }, []);

  const disableImageGeneration = useCallback((errorCode: string) => {
    setIsImageGenerationDisabled(true);
    showErrorModal({ code: errorCode });
  }, []);

  return {
    lightboxImage,
    openLightbox,
    closeLightbox,
    fontSize,
    increaseFontSize,
    decreaseFontSize,
    resetFontSize,
    debugMode,
    toggleDebugMode,
    animationEnabled,
    setAnimationEnabled,
    textEffectsEnabled,
    setTextEffectsEnabled,
    useNeutralTheme,
    setUseNeutralTheme,
    isImageGenerationDisabled,
    disableImageGeneration,
  };
}
