/**
 * Theme Test Panel
 * 
 * TEMPORARY: For testing theme settings. Remove when done.
 * Allows live modification of theme configuration.
 * Changes are applied immediately when user modifies any setting.
 */

import { useState, useEffect, useCallback, useRef } from 'react';
import {
  Drawer,
  Stack,
  Select,
  Switch,
  TextInput,
  Button,
  Text,
  Divider,
  ActionIcon,
  Tooltip,
  ScrollArea,
  Group,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { IconPalette } from '@tabler/icons-react';
import type { 
  PartialGameTheme, 
  CornerStyle, 
  ThemeColor, 
  BackgroundTint, 
  BackgroundAnimation,
  PlayerIndicator, 
  ThinkingStyle, 
  StreamingCursor,
  MessageFont, 
  CardBgColor,
  FontColor,
  CardBorderThickness,
  DividerStyle,
} from '../theme/types';
import { PRESET_THEMES } from '../theme/defaults';

const CORNER_STYLES: { value: CornerStyle; label: string }[] = [
  { value: 'brackets', label: '[ ] Brackets' },
  { value: 'flourish', label: '❧ Flourish' },
  { value: 'arrows', label: '▶ Arrows' },
  { value: 'dots', label: '••• Dots' },
  { value: 'dot', label: '• Dot' },
  { value: 'cursor', label: '_ Cursor' },
  { value: 'none', label: 'None' },
];

const THEME_COLORS: { value: ThemeColor; label: string }[] = [
  { value: 'amber', label: 'Amber (Gold)' },
  { value: 'emerald', label: 'Emerald (Green)' },
  { value: 'cyan', label: 'Cyan (Teal)' },
  { value: 'violet', label: 'Violet (Purple)' },
  { value: 'rose', label: 'Rose (Red)' },
  { value: 'slate', label: 'Slate (Gray)' },
  { value: 'hacker', label: 'Hacker (Neon Green)' },
  { value: 'terminal', label: 'Terminal (Neon Red)' },
  { value: 'brown', label: 'Brown (Dark)' },
  { value: 'brownLight', label: 'Brown (Light/Tan)' },
  { value: 'pink', label: 'Pink (Vibrant)' },
  { value: 'pinkLight', label: 'Pink (Soft/Pastel)' },
  { value: 'orange', label: 'Orange (Vibrant)' },
  { value: 'orangeLight', label: 'Orange (Soft/Warm)' },
];

const CARD_BG_COLORS: { value: CardBgColor; label: string }[] = [
  // Neutrals
  { value: 'white', label: 'White' },
  { value: 'creme', label: 'Creme' },
  { value: 'dark', label: 'Dark' },
  { value: 'black', label: 'Black' },
  // Colors (Dark)
  { value: 'blue', label: 'Blue (Dark)' },
  { value: 'green', label: 'Green (Hacker)' },
  { value: 'red', label: 'Red (Terminal)' },
  { value: 'amber', label: 'Amber (Dark)' },
  { value: 'violet', label: 'Violet (Dark)' },
  { value: 'rose', label: 'Rose (Dark)' },
  { value: 'cyan', label: 'Cyan (Dark)' },
  // Colors (Light)
  { value: 'blueLight', label: 'Blue (Light)' },
  { value: 'greenLight', label: 'Green (Light)' },
  { value: 'redLight', label: 'Red (Light)' },
  { value: 'amberLight', label: 'Amber (Light)' },
  { value: 'violetLight', label: 'Violet (Light)' },
  { value: 'roseLight', label: 'Rose (Light)' },
  { value: 'cyanLight', label: 'Cyan (Light)' },
  { value: 'pinkLight', label: 'Pink (Light)' },
  { value: 'orangeLight', label: 'Orange (Light)' },
  // Colors (Dark - continued)
  { value: 'pink', label: 'Pink (Dark)' },
  { value: 'orange', label: 'Orange (Dark)' },
];

const FONT_COLORS: { value: FontColor; label: string }[] = [
  { value: 'dark', label: 'Dark' },
  { value: 'light', label: 'Light' },
  { value: 'hacker', label: 'Hacker Green' },
  { value: 'terminal', label: 'Terminal Red' },
  { value: 'pink', label: 'Pink' },
  { value: 'amber', label: 'Amber' },
  { value: 'cyan', label: 'Cyan' },
  { value: 'violet', label: 'Violet' },
];


const BORDER_THICKNESSES: { value: CardBorderThickness; label: string }[] = [
  { value: 'none', label: 'None' },
  { value: 'thin', label: 'Thin' },
  { value: 'medium', label: 'Medium' },
  { value: 'thick', label: 'Thick' },
];


const BACKGROUND_TINTS: { value: BackgroundTint; label: string }[] = [
  // Light
  { value: 'neutral', label: 'Neutral' },
  { value: 'warm', label: 'Warm' },
  { value: 'cool', label: 'Cool' },
  { value: 'pink', label: 'Pink (Light)' },
  { value: 'green', label: 'Green (Light)' },
  { value: 'blue', label: 'Blue (Light)' },
  { value: 'violet', label: 'Violet (Light)' },
  // Dark
  { value: 'dark', label: 'Dark (Gray)' },
  { value: 'black', label: 'Black' },
  { value: 'darkCyan', label: 'Dark Cyan' },
  { value: 'darkViolet', label: 'Dark Violet' },
  { value: 'darkBlue', label: 'Dark Blue' },
  { value: 'darkRose', label: 'Dark Rose' },
];

const BACKGROUND_ANIMATIONS: { value: BackgroundAnimation; label: string }[] = [
  { value: 'none', label: 'None' },
  { value: 'stars', label: '✨ Stars (Space/Sci-Fi)' },
  { value: 'bubbles', label: '🫧 Bubbles (Ocean)' },
  { value: 'fireflies', label: '🪲 Fireflies (Fantasy)' },
  { value: 'snow', label: '❄️ Snow' },
  { value: 'rain', label: '🌧️ Rain (Horror)' },
  { value: 'matrix', label: '💻 Matrix (Hacker)' },
];

const PLAYER_INDICATORS: { value: PlayerIndicator; label: string }[] = [
  { value: 'dot', label: '• Dot' },
  { value: 'chevron', label: '> Chevron' },
  { value: 'pipe', label: '| Pipe' },
  { value: 'cursor', label: '▌ Cursor' },
  { value: 'underscore', label: '_ Underscore' },
  { value: 'diamond', label: '◆ Diamond' },
  { value: 'arrow', label: '→ Arrow' },
  { value: 'star', label: '★ Star' },
  { value: 'none', label: 'None' },
];

const THINKING_STYLES: { value: ThinkingStyle; label: string }[] = [
  { value: 'dots', label: 'Dots' },
  { value: 'spinner', label: 'Spinner' },
  { value: 'pulse', label: 'Pulse' },
  { value: 'typewriter', label: 'Typewriter' },
];

const STREAMING_CURSORS: { value: StreamingCursor; label: string }[] = [
  { value: 'dots', label: '... Dots (Animated)' },
  { value: 'block', label: '█ Block' },
  { value: 'pipe', label: '| Pipe' },
  { value: 'underscore', label: '_ Underscore' },
  { value: 'none', label: 'None' },
];

const MESSAGE_FONTS: { value: MessageFont; label: string }[] = [
  { value: 'serif', label: 'Serif' },
  { value: 'sans', label: 'Sans' },
  { value: 'mono', label: 'Mono' },
  { value: 'fantasy', label: 'Fantasy' },
];

const DIVIDER_STYLES: { value: DividerStyle; label: string }[] = [
  { value: 'dot', label: '• Dot' },
  { value: 'dots', label: '• • • Dots' },
  { value: 'line', label: '— Line' },
  { value: 'diamond', label: '◆ Diamond' },
  { value: 'star', label: '✦ Star' },
  { value: 'dash', label: '--- Dash' },
  { value: 'none', label: 'None' },
];

const PRESET_OPTIONS = [
  { value: '', label: '-- Select Preset --' },
  { value: 'default', label: '⭐ Default (Neutral)' },
  { value: 'minimal', label: 'Minimal / Clean' },
  { value: 'fantasy', label: 'Fantasy / Medieval' },
  { value: 'scifi', label: 'Sci-Fi / Cyberpunk' },
  { value: 'horror', label: 'Horror / Mystery' },
  { value: 'adventure', label: 'Adventure / Exploration' },
  { value: 'mystery', label: 'Mystery / Mystic' },
  { value: 'detective', label: 'Detective / Noir' },
  { value: 'space', label: 'Space / Cosmic' },
  { value: 'terminal', label: 'Terminal (Green/Black)' },
  { value: 'hacker', label: 'Hacker (Aggressive Red/Green)' },
  { value: 'playful', label: 'Playful / Kids' },
  { value: 'barbie', label: 'Barbie / Pink Dream' },
  { value: 'nature', label: 'Nature / Forest' },
  { value: 'ocean', label: 'Ocean / Underwater' },
  { value: 'retro', label: 'Retro / 80s' },
  { value: 'western', label: 'Western / Wild West' },
];

interface ThemeTestPanelProps {
  currentTheme: PartialGameTheme | undefined;
  onThemeChange: (theme: PartialGameTheme) => void;
}

export function ThemeTestPanel({ currentTheme, onThemeChange }: ThemeTestPanelProps) {
  const [opened, { open, close }] = useDisclosure(false);
  
  // Local state for all theme values (using backend defaults)
  const [cornerStyle, setCornerStyle] = useState<CornerStyle>(currentTheme?.corners?.style || 'none');
  const [cornerColor, setCornerColor] = useState<ThemeColor>(currentTheme?.corners?.color || 'amber');
  const [cornerTopLeft, setCornerTopLeft] = useState(currentTheme?.corners?.positions?.topLeft ?? true);
  const [cornerTopRight, setCornerTopRight] = useState(currentTheme?.corners?.positions?.topRight ?? false);
  const [cornerBottomLeft, setCornerBottomLeft] = useState(currentTheme?.corners?.positions?.bottomLeft ?? false);
  const [cornerBottomRight, setCornerBottomRight] = useState(currentTheme?.corners?.positions?.bottomRight ?? true);
  const [cornerBlink, setCornerBlink] = useState(currentTheme?.corners?.blink ?? false);
  const [bgTint, setBgTint] = useState<BackgroundTint>(currentTheme?.background?.tint || 'neutral');
  const [bgAnimation, setBgAnimation] = useState<BackgroundAnimation>(currentTheme?.background?.animation || 'none');
  
  // Player message settings
  const [playerColor, setPlayerColor] = useState<ThemeColor>(currentTheme?.player?.color || 'cyan');
  const [playerIndicator, setPlayerIndicator] = useState<PlayerIndicator>(currentTheme?.player?.indicator || 'chevron');
  const [indicatorBlink, setIndicatorBlink] = useState(currentTheme?.player?.indicatorBlink ?? false);
  const [playerBgColor, setPlayerBgColor] = useState<CardBgColor>(currentTheme?.player?.bgColor || 'white');
  const [playerFontColor, setPlayerFontColor] = useState<FontColor>(currentTheme?.player?.fontColor || 'dark');
  
  // Game message settings
  const [dropCap, setDropCap] = useState(currentTheme?.gameMessage?.dropCap ?? true);
  const [dropCapColor, setDropCapColor] = useState<ThemeColor>(currentTheme?.gameMessage?.dropCapColor || 'amber');
  const [gameBgColor, setGameBgColor] = useState<CardBgColor>(currentTheme?.gameMessage?.bgColor || 'white');
  const [gameFontColor, setGameFontColor] = useState<FontColor>(currentTheme?.gameMessage?.fontColor || 'dark');
  
  // Shared card settings
  const [borderThickness, setBorderThickness] = useState<CardBorderThickness>(currentTheme?.cards?.borderThickness || 'thin');
  
  // Per-message border colors
  const [playerBorderColor, setPlayerBorderColor] = useState<ThemeColor>(currentTheme?.player?.borderColor || 'cyan');
  const [gameBorderColor, setGameBorderColor] = useState<ThemeColor>(currentTheme?.gameMessage?.borderColor || 'amber');
  
  // Status field settings
  const [statusBgColor, setStatusBgColor] = useState<CardBgColor>(currentTheme?.statusFields?.bgColor || 'creme');
  const [statusAccentColor, setStatusAccentColor] = useState<ThemeColor>(currentTheme?.statusFields?.accentColor || 'amber');
  const [statusBorderColor, setStatusBorderColor] = useState<ThemeColor>(currentTheme?.statusFields?.borderColor || 'amber');
  const [statusFontColor, setStatusFontColor] = useState<FontColor>(currentTheme?.statusFields?.fontColor || 'dark');
  
  // Header settings
  const [headerBgColor, setHeaderBgColor] = useState<CardBgColor>(currentTheme?.header?.bgColor || 'white');
  const [headerFontColor, setHeaderFontColor] = useState<FontColor>(currentTheme?.header?.fontColor || 'dark');
  const [headerAccentColor, setHeaderAccentColor] = useState<ThemeColor>(currentTheme?.header?.accentColor || 'amber');
  
  // Divider settings
  const [dividerStyle, setDividerStyle] = useState<DividerStyle>(currentTheme?.divider?.style || 'dot');
  const [dividerColor, setDividerColor] = useState<ThemeColor>(currentTheme?.divider?.color || 'amber');
  
  // Thinking and typography
  const [thinkingText, setThinkingText] = useState(currentTheme?.thinking?.text || 'The story unfolds...');
  const [thinkingStyle, setThinkingStyle] = useState<ThinkingStyle>(currentTheme?.thinking?.style || 'dots');
  const [streamingCursor, setStreamingCursor] = useState<StreamingCursor>(currentTheme?.thinking?.streamingCursor || 'dots');
  const [messageFont, setMessageFont] = useState<MessageFont>(currentTheme?.typography?.messages || 'sans');

  // Track whether we're syncing from props (don't trigger onThemeChange during sync)
  const isSyncingRef = useRef(false);
  const currentThemeJson = currentTheme ? JSON.stringify(currentTheme) : null;
  const prevThemeJsonRef = useRef<string | null>(null);
  
  // Sync local state when currentTheme changes externally (e.g., when AI generates a theme)
  /* eslint-disable @eslint-react/hooks-extra/no-direct-set-state-in-use-effect -- Intentional: sync external theme to local state */
  useEffect(() => {
    if (!currentTheme || currentThemeJson === prevThemeJsonRef.current) return;
    prevThemeJsonRef.current = currentThemeJson;
    
    // Mark that we're syncing - this prevents auto-apply from triggering onThemeChange
    isSyncingRef.current = true;
    
    // Batch all state updates - React 18+ batches these automatically
    if (currentTheme.corners?.style) setCornerStyle(currentTheme.corners.style);
    if (currentTheme.corners?.color) setCornerColor(currentTheme.corners.color);
    if (currentTheme.corners?.positions?.topLeft !== undefined) setCornerTopLeft(currentTheme.corners.positions.topLeft);
    if (currentTheme.corners?.positions?.topRight !== undefined) setCornerTopRight(currentTheme.corners.positions.topRight);
    if (currentTheme.corners?.positions?.bottomLeft !== undefined) setCornerBottomLeft(currentTheme.corners.positions.bottomLeft);
    if (currentTheme.corners?.positions?.bottomRight !== undefined) setCornerBottomRight(currentTheme.corners.positions.bottomRight);
    if (currentTheme.corners?.blink !== undefined) setCornerBlink(currentTheme.corners.blink);
    if (currentTheme.background?.tint) setBgTint(currentTheme.background.tint);
    if (currentTheme.background?.animation) setBgAnimation(currentTheme.background.animation);
    if (currentTheme.player?.color) setPlayerColor(currentTheme.player.color);
    if (currentTheme.player?.indicator) setPlayerIndicator(currentTheme.player.indicator);
    if (currentTheme.player?.indicatorBlink !== undefined) setIndicatorBlink(currentTheme.player.indicatorBlink);
    if (currentTheme.player?.bgColor) setPlayerBgColor(currentTheme.player.bgColor);
    if (currentTheme.player?.fontColor) setPlayerFontColor(currentTheme.player.fontColor);
    if (currentTheme.player?.borderColor) setPlayerBorderColor(currentTheme.player.borderColor);
    if (currentTheme.gameMessage?.dropCap !== undefined) setDropCap(currentTheme.gameMessage.dropCap);
    if (currentTheme.gameMessage?.dropCapColor) setDropCapColor(currentTheme.gameMessage.dropCapColor);
    if (currentTheme.gameMessage?.bgColor) setGameBgColor(currentTheme.gameMessage.bgColor);
    if (currentTheme.gameMessage?.fontColor) setGameFontColor(currentTheme.gameMessage.fontColor);
    if (currentTheme.gameMessage?.borderColor) setGameBorderColor(currentTheme.gameMessage.borderColor);
    if (currentTheme.cards?.borderThickness) setBorderThickness(currentTheme.cards.borderThickness);
    if (currentTheme.thinking?.text) setThinkingText(currentTheme.thinking.text);
    if (currentTheme.thinking?.style) setThinkingStyle(currentTheme.thinking.style);
    if (currentTheme.thinking?.streamingCursor) setStreamingCursor(currentTheme.thinking.streamingCursor);
    if (currentTheme.typography?.messages) setMessageFont(currentTheme.typography.messages);
    if (currentTheme.statusFields?.bgColor) setStatusBgColor(currentTheme.statusFields.bgColor);
    if (currentTheme.statusFields?.accentColor) setStatusAccentColor(currentTheme.statusFields.accentColor);
    if (currentTheme.statusFields?.borderColor) setStatusBorderColor(currentTheme.statusFields.borderColor);
    if (currentTheme.statusFields?.fontColor) setStatusFontColor(currentTheme.statusFields.fontColor);
    if (currentTheme.header?.bgColor) setHeaderBgColor(currentTheme.header.bgColor);
    if (currentTheme.header?.fontColor) setHeaderFontColor(currentTheme.header.fontColor);
    if (currentTheme.header?.accentColor) setHeaderAccentColor(currentTheme.header.accentColor);
    if (currentTheme.divider?.style) setDividerStyle(currentTheme.divider.style);
    if (currentTheme.divider?.color) setDividerColor(currentTheme.divider.color);
    
    // Reset sync flag after React processes all updates
    requestAnimationFrame(() => {
      isSyncingRef.current = false;
    });
  }, [currentThemeJson, currentTheme]);

  // Build and apply theme whenever any value changes
  const buildTheme = useCallback((): PartialGameTheme => ({
    corners: { 
      style: cornerStyle, 
      color: cornerColor,
      positions: {
        topLeft: cornerTopLeft,
        topRight: cornerTopRight,
        bottomLeft: cornerBottomLeft,
        bottomRight: cornerBottomRight,
      },
      blink: cornerBlink,
    },
    background: { tint: bgTint, animation: bgAnimation },
    player: {
      color: playerColor,
      indicator: playerIndicator,
      indicatorBlink: indicatorBlink,
      bgColor: playerBgColor,
      fontColor: playerFontColor,
      borderColor: playerBorderColor,
    },
    gameMessage: {
      dropCap: dropCap,
      dropCapColor: dropCapColor,
      bgColor: gameBgColor,
      fontColor: gameFontColor,
      borderColor: gameBorderColor,
    },
    cards: {
      borderThickness: borderThickness,
    },
    thinking: {
      text: thinkingText,
      style: thinkingStyle,
      streamingCursor: streamingCursor,
    },
    typography: {
      messages: messageFont,
    },
    statusFields: {
      bgColor: statusBgColor,
      accentColor: statusAccentColor,
      borderColor: statusBorderColor,
      fontColor: statusFontColor,
    },
    header: {
      bgColor: headerBgColor,
      fontColor: headerFontColor,
      accentColor: headerAccentColor,
    },
    divider: {
      style: dividerStyle,
      color: dividerColor,
    },
  }), [cornerStyle, cornerColor, cornerTopLeft, cornerTopRight, cornerBottomLeft, cornerBottomRight, cornerBlink, bgTint, bgAnimation, playerColor, playerIndicator, indicatorBlink, playerBgColor, playerFontColor, playerBorderColor, dropCap, dropCapColor, gameBgColor, gameFontColor, gameBorderColor, borderThickness, thinkingText, thinkingStyle, streamingCursor, messageFont, statusBgColor, statusAccentColor, statusBorderColor, statusFontColor, headerBgColor, headerFontColor, headerAccentColor, dividerStyle, dividerColor]);

  // Skip the very first mount to prevent overriding AI theme with defaults
  const isFirstMountRef = useRef(true);
  
  // Auto-apply theme on any change (but skip initial mount and sync from props)
  useEffect(() => {
    // Skip first mount - don't override with defaults before AI theme arrives
    if (isFirstMountRef.current) {
      isFirstMountRef.current = false;
      return;
    }
    // Skip if we're syncing from props - don't set themeOverride, let state.theme flow directly
    if (isSyncingRef.current) {
      return;
    }
    onThemeChange(buildTheme());
  }, [buildTheme, onThemeChange]);

  const loadPreset = (presetName: string) => {
    if (!presetName) return;
    const preset = PRESET_THEMES[presetName];
    if (!preset) return;

    if (preset.corners?.style) setCornerStyle(preset.corners.style);
    if (preset.corners?.color) setCornerColor(preset.corners.color);
    if (preset.corners?.positions) {
      setCornerTopLeft(preset.corners.positions.topLeft);
      setCornerTopRight(preset.corners.positions.topRight);
      setCornerBottomLeft(preset.corners.positions.bottomLeft);
      setCornerBottomRight(preset.corners.positions.bottomRight);
    }
    if (preset.corners?.blink !== undefined) setCornerBlink(preset.corners.blink);
    if (preset.background?.tint) setBgTint(preset.background.tint);
    if (preset.background?.animation) setBgAnimation(preset.background.animation);
    if (preset.player?.color) setPlayerColor(preset.player.color);
    if (preset.player?.indicator) setPlayerIndicator(preset.player.indicator);
    if (preset.player?.indicatorBlink !== undefined) setIndicatorBlink(preset.player.indicatorBlink);
    if (preset.player?.bgColor) setPlayerBgColor(preset.player.bgColor);
    if (preset.player?.fontColor) setPlayerFontColor(preset.player.fontColor);
    if (preset.gameMessage?.dropCap !== undefined) setDropCap(preset.gameMessage.dropCap);
    if (preset.gameMessage?.dropCapColor) setDropCapColor(preset.gameMessage.dropCapColor);
    if (preset.gameMessage?.bgColor) setGameBgColor(preset.gameMessage.bgColor);
    if (preset.gameMessage?.fontColor) setGameFontColor(preset.gameMessage.fontColor);
    if (preset.gameMessage?.borderColor) setGameBorderColor(preset.gameMessage.borderColor);
    if (preset.player?.borderColor) setPlayerBorderColor(preset.player.borderColor);
    if (preset.cards?.borderThickness) setBorderThickness(preset.cards.borderThickness);
    if (preset.thinking?.text) setThinkingText(preset.thinking.text);
    if (preset.thinking?.style) setThinkingStyle(preset.thinking.style);
    if (preset.thinking?.streamingCursor) setStreamingCursor(preset.thinking.streamingCursor);
    if (preset.typography?.messages) setMessageFont(preset.typography.messages);
    if (preset.statusFields?.bgColor) setStatusBgColor(preset.statusFields.bgColor);
    if (preset.statusFields?.accentColor) setStatusAccentColor(preset.statusFields.accentColor);
    if (preset.statusFields?.borderColor) setStatusBorderColor(preset.statusFields.borderColor);
    if (preset.statusFields?.fontColor) setStatusFontColor(preset.statusFields.fontColor);
    if (preset.header?.bgColor) setHeaderBgColor(preset.header.bgColor);
    if (preset.header?.fontColor) setHeaderFontColor(preset.header.fontColor);
    if (preset.header?.accentColor) setHeaderAccentColor(preset.header.accentColor);
    if (preset.divider?.style) setDividerStyle(preset.divider.style);
    if (preset.divider?.color) setDividerColor(preset.divider.color);
  };

  const logCurrentTheme = () => {
    console.log('[ThemeTestPanel] Current theme config:');
    console.log(JSON.stringify(buildTheme(), null, 2));
  };

  return (
    <>
      <Tooltip label="Theme Tester" position="bottom">
        <ActionIcon
          variant="subtle"
          color="gray"
          onClick={open}
          aria-label="Open theme tester"
          size="lg"
        >
          <IconPalette size={18} />
        </ActionIcon>
      </Tooltip>

      <Drawer
        opened={opened}
        onClose={close}
        title="Theme Tester"
        position="right"
        size="sm"
        styles={{ body: { padding: 0 } }}
      >
        <ScrollArea h="calc(100vh - 80px)" px="md" pb="md">
          <Stack gap="md">
            {/* Presets */}
            <Select
              label="Load Preset"
              data={PRESET_OPTIONS}
              onChange={(v) => v && loadPreset(v)}
              placeholder="Select a preset..."
            />

            <Divider label="Background" labelPosition="center" />
            
            <Select
              label="Game Player Background"
              data={BACKGROUND_TINTS}
              value={bgTint}
              onChange={(v) => v && setBgTint(v as BackgroundTint)}
            />
            <Select
              label="Background Animation"
              description="Animated particle effects"
              data={BACKGROUND_ANIMATIONS}
              value={bgAnimation}
              onChange={(v) => v && setBgAnimation(v as BackgroundAnimation)}
            />

            <Divider label="Card Styling (Shared)" labelPosition="center" />
            
            <Select
              label="Border Thickness"
              data={BORDER_THICKNESSES}
              value={borderThickness}
              onChange={(v) => v && setBorderThickness(v as CardBorderThickness)}
            />

            <Divider label="Player Messages" labelPosition="center" />
            
            <Select
              label="Player BG Color"
              data={CARD_BG_COLORS}
              value={playerBgColor}
              onChange={(v) => v && setPlayerBgColor(v as CardBgColor)}
            />
            <Select
              label="Player Font Color"
              data={FONT_COLORS}
              value={playerFontColor}
              onChange={(v) => v && setPlayerFontColor(v as FontColor)}
            />
            <Select
              label="Player Accent Color"
              data={THEME_COLORS}
              value={playerColor}
              onChange={(v) => v && setPlayerColor(v as ThemeColor)}
            />
            <Select
              label="Player Indicator"
              data={PLAYER_INDICATORS}
              value={playerIndicator}
              onChange={(v) => v && setPlayerIndicator(v as PlayerIndicator)}
            />
            <Switch
              label="Blink Indicator"
              checked={indicatorBlink}
              onChange={(e) => setIndicatorBlink(e.currentTarget.checked)}
            />
            <Select
              label="Player Border Color"
              data={THEME_COLORS}
              value={playerBorderColor}
              onChange={(v) => v && setPlayerBorderColor(v as ThemeColor)}
            />

            <Divider label="AI/Game Messages" labelPosition="center" />
            
            <Select
              label="AI BG Color"
              data={CARD_BG_COLORS}
              value={gameBgColor}
              onChange={(v) => v && setGameBgColor(v as CardBgColor)}
            />
            <Select
              label="AI Font Color"
              data={FONT_COLORS}
              value={gameFontColor}
              onChange={(v) => v && setGameFontColor(v as FontColor)}
            />
            <Select
              label="AI Border Color"
              data={THEME_COLORS}
              value={gameBorderColor}
              onChange={(v) => v && setGameBorderColor(v as ThemeColor)}
            />
            <Switch
              label="Drop Cap"
              checked={dropCap}
              onChange={(e) => setDropCap(e.currentTarget.checked)}
            />
            <Select
              label="Drop Cap Color"
              data={THEME_COLORS}
              value={dropCapColor}
              onChange={(v) => v && setDropCapColor(v as ThemeColor)}
              disabled={!dropCap}
            />
            
            <Divider label="AI Corner Decorations" labelPosition="center" />
            
            <Select
              label="Corner Style"
              data={CORNER_STYLES}
              value={cornerStyle}
              onChange={(v) => v && setCornerStyle(v as CornerStyle)}
            />
            <Select
              label="Corner Color"
              data={THEME_COLORS}
              value={cornerColor}
              onChange={(v) => v && setCornerColor(v as ThemeColor)}
            />
            <Group grow>
              <Switch
                label="Top Left"
                checked={cornerTopLeft}
                onChange={(e) => setCornerTopLeft(e.currentTarget.checked)}
                size="xs"
              />
              <Switch
                label="Top Right"
                checked={cornerTopRight}
                onChange={(e) => setCornerTopRight(e.currentTarget.checked)}
                size="xs"
              />
            </Group>
            <Group grow>
              <Switch
                label="Bottom Left"
                checked={cornerBottomLeft}
                onChange={(e) => setCornerBottomLeft(e.currentTarget.checked)}
                size="xs"
              />
              <Switch
                label="Bottom Right"
                checked={cornerBottomRight}
                onChange={(e) => setCornerBottomRight(e.currentTarget.checked)}
                size="xs"
              />
            </Group>
            <Switch
              label="Blink Corners"
              checked={cornerBlink}
              onChange={(e) => setCornerBlink(e.currentTarget.checked)}
            />

            <Divider label="Status Fields" labelPosition="center" />
            
            <Select
              label="Status BG Color"
              data={CARD_BG_COLORS}
              value={statusBgColor}
              onChange={(v) => v && setStatusBgColor(v as CardBgColor)}
            />
            <Select
              label="Status Accent Color"
              data={THEME_COLORS}
              value={statusAccentColor}
              onChange={(v) => v && setStatusAccentColor(v as ThemeColor)}
            />
            <Select
              label="Status Border Color"
              data={THEME_COLORS}
              value={statusBorderColor}
              onChange={(v) => v && setStatusBorderColor(v as ThemeColor)}
            />
            <Select
              label="Status Font Color"
              data={FONT_COLORS}
              value={statusFontColor}
              onChange={(v) => v && setStatusFontColor(v as FontColor)}
            />

            <Divider label="Header Bar" labelPosition="center" />
            
            <Select
              label="Header BG Color"
              data={CARD_BG_COLORS}
              value={headerBgColor}
              onChange={(v) => v && setHeaderBgColor(v as CardBgColor)}
            />
            <Select
              label="Header Font Color"
              data={FONT_COLORS}
              value={headerFontColor}
              onChange={(v) => v && setHeaderFontColor(v as FontColor)}
            />
            <Select
              label="Header Accent Color"
              data={THEME_COLORS}
              value={headerAccentColor}
              onChange={(v) => v && setHeaderAccentColor(v as ThemeColor)}
            />

            <Divider label="Message Divider" labelPosition="center" />
            
            <Select
              label="Divider Style"
              data={DIVIDER_STYLES}
              value={dividerStyle}
              onChange={(v) => v && setDividerStyle(v as DividerStyle)}
            />
            <Select
              label="Divider Color"
              data={THEME_COLORS}
              value={dividerColor}
              onChange={(v) => v && setDividerColor(v as ThemeColor)}
            />

            <Divider label="Thinking Indicator" labelPosition="center" />
            
            <TextInput
              label="Thinking Text"
              value={thinkingText}
              onChange={(e) => setThinkingText(e.currentTarget.value)}
            />
            <Select
              label="Thinking Style"
              data={THINKING_STYLES}
              value={thinkingStyle}
              onChange={(v) => v && setThinkingStyle(v as ThinkingStyle)}
            />
            <Select
              label="Streaming Cursor"
              description="Cursor shown while AI generates text"
              data={STREAMING_CURSORS}
              value={streamingCursor}
              onChange={(v) => v && setStreamingCursor(v as StreamingCursor)}
            />

            <Divider label="Typography" labelPosition="center" />
            
            <Select
              label="Message Font"
              data={MESSAGE_FONTS}
              value={messageFont}
              onChange={(v) => v && setMessageFont(v as MessageFont)}
            />

            <Divider />

            <Button variant="subtle" onClick={logCurrentTheme} size="xs">
              Log to Console
            </Button>

            <Text size="xs" c="dimmed" ta="center">
              This panel is for testing only. Changes apply immediately.
            </Text>
          </Stack>
        </ScrollArea>
      </Drawer>
    </>
  );
}
