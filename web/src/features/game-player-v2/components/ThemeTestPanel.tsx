/**
 * Theme Test Panel
 * 
 * TEMPORARY: For testing theme settings. Remove when done.
 * Allows live modification of theme configuration.
 */

import { useState } from 'react';
import {
  Drawer,
  Stack,
  Select,
  Switch,
  TextInput,
  Button,
  Group,
  Text,
  Divider,
  ActionIcon,
  Tooltip,
  ScrollArea,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { IconPalette } from '@tabler/icons-react';
import type { PartialGameTheme, CornerStyle, ThemeColor, BackgroundAnimation, BackgroundTint, PlayerIndicator, ThinkingStyle, MessageFont, PlayerBgColor } from '../theme/types';
import { PRESET_THEMES } from '../theme/defaults';

const CORNER_STYLES = [
  { value: 'brackets', label: 'Brackets' },
  { value: 'flourish', label: 'Flourish' },
  { value: 'arrows', label: 'Arrows' },
  { value: 'dots', label: 'Dots' },
  { value: 'none', label: 'None' },
];

const THEME_COLORS = [
  { value: 'amber', label: 'Amber' },
  { value: 'emerald', label: 'Emerald' },
  { value: 'cyan', label: 'Cyan' },
  { value: 'violet', label: 'Violet' },
  { value: 'rose', label: 'Rose' },
  { value: 'slate', label: 'Slate' },
];

const PLAYER_BG_COLORS = [
  { value: 'cyan', label: 'Cyan' },
  { value: 'amber', label: 'Amber' },
  { value: 'violet', label: 'Violet' },
  { value: 'slate', label: 'Slate' },
  { value: 'white', label: 'White' },
  { value: 'emerald', label: 'Emerald' },
  { value: 'rose', label: 'Rose' },
];

const BACKGROUND_ANIMATIONS = [
  { value: 'none', label: 'None' },
  { value: 'stars', label: 'Stars' },
  { value: 'rain', label: 'Rain' },
  { value: 'fog', label: 'Fog' },
  { value: 'particles', label: 'Particles' },
  { value: 'scanlines', label: 'Scanlines' },
];

const BACKGROUND_TINTS = [
  { value: 'warm', label: 'Warm' },
  { value: 'cool', label: 'Cool' },
  { value: 'neutral', label: 'Neutral' },
  { value: 'dark', label: 'Dark' },
];

const PLAYER_INDICATORS = [
  { value: 'dot', label: 'Dot' },
  { value: 'arrow', label: 'Arrow' },
  { value: 'chevron', label: 'Chevron' },
  { value: 'diamond', label: 'Diamond' },
  { value: 'none', label: 'None' },
];

const THINKING_STYLES = [
  { value: 'dots', label: 'Dots' },
  { value: 'spinner', label: 'Spinner' },
  { value: 'pulse', label: 'Pulse' },
  { value: 'typewriter', label: 'Typewriter' },
];

const MESSAGE_FONTS = [
  { value: 'serif', label: 'Serif' },
  { value: 'sans', label: 'Sans' },
  { value: 'mono', label: 'Mono' },
  { value: 'fantasy', label: 'Fantasy' },
];

const PRESET_OPTIONS = [
  { value: '', label: '-- Select Preset --' },
  { value: 'scifi', label: 'Sci-Fi' },
  { value: 'fantasy', label: 'Fantasy' },
  { value: 'horror', label: 'Horror' },
  { value: 'adventure', label: 'Adventure' },
  { value: 'mystery', label: 'Mystery' },
  { value: 'space', label: 'Space' },
];

interface ThemeTestPanelProps {
  currentTheme: PartialGameTheme | undefined;
  onThemeChange: (theme: PartialGameTheme) => void;
}

export function ThemeTestPanel({ currentTheme, onThemeChange }: ThemeTestPanelProps) {
  const [opened, { open, close }] = useDisclosure(false);
  
  // Local state for all theme values
  const [cornerStyle, setCornerStyle] = useState(currentTheme?.corners?.style || 'brackets');
  const [cornerColor, setCornerColor] = useState(currentTheme?.corners?.color || 'amber');
  const [bgAnimation, setBgAnimation] = useState(currentTheme?.background?.animation || 'none');
  const [bgTint, setBgTint] = useState(currentTheme?.background?.tint || 'warm');
  const [playerColor, setPlayerColor] = useState(currentTheme?.player?.color || 'cyan');
  const [playerIndicator, setPlayerIndicator] = useState(currentTheme?.player?.indicator || 'dot');
  const [playerShowChevron, setPlayerShowChevron] = useState(currentTheme?.player?.showChevron ?? false);
  const [playerBgColor, setPlayerBgColor] = useState(currentTheme?.player?.bgColor || 'cyan');
  const [dropCap, setDropCap] = useState(currentTheme?.gameMessage?.dropCap ?? true);
  const [dropCapColor, setDropCapColor] = useState(currentTheme?.gameMessage?.dropCapColor || 'amber');
  const [thinkingText, setThinkingText] = useState(currentTheme?.thinking?.text || 'The story unfolds...');
  const [thinkingStyle, setThinkingStyle] = useState(currentTheme?.thinking?.style || 'dots');
  const [messageFont, setMessageFont] = useState(currentTheme?.typography?.messages || 'sans');

  const applyTheme = () => {
    const theme: PartialGameTheme = {
      corners: { style: cornerStyle as any, color: cornerColor as any },
      background: { animation: bgAnimation as any, tint: bgTint as any },
      player: {
        color: playerColor as any,
        indicator: playerIndicator as any,
        showChevron: playerShowChevron,
        bgColor: playerBgColor as any,
      },
      gameMessage: {
        dropCap: dropCap,
        dropCapColor: dropCapColor as any,
      },
      thinking: {
        text: thinkingText,
        style: thinkingStyle as any,
      },
      typography: {
        messages: messageFont as any,
      },
    };
    onThemeChange(theme);
  };

  const loadPreset = (presetName: string) => {
    if (!presetName) return;
    const preset = PRESET_THEMES[presetName];
    if (!preset) return;

    if (preset.corners?.style) setCornerStyle(preset.corners.style);
    if (preset.corners?.color) setCornerColor(preset.corners.color);
    if (preset.background?.animation) setBgAnimation(preset.background.animation);
    if (preset.background?.tint) setBgTint(preset.background.tint);
    if (preset.player?.color) setPlayerColor(preset.player.color);
    if (preset.player?.indicator) setPlayerIndicator(preset.player.indicator);
    if (preset.player?.showChevron !== undefined) setPlayerShowChevron(preset.player.showChevron);
    if (preset.player?.bgColor) setPlayerBgColor(preset.player.bgColor);
    if (preset.gameMessage?.dropCap !== undefined) setDropCap(preset.gameMessage.dropCap);
    if (preset.gameMessage?.dropCapColor) setDropCapColor(preset.gameMessage.dropCapColor);
    if (preset.thinking?.text) setThinkingText(preset.thinking.text);
    if (preset.thinking?.style) setThinkingStyle(preset.thinking.style);
    if (preset.typography?.messages) setMessageFont(preset.typography.messages);
  };

  const logCurrentTheme = () => {
    const theme: PartialGameTheme = {
      corners: { style: cornerStyle as any, color: cornerColor as any },
      background: { animation: bgAnimation as any, tint: bgTint as any },
      player: {
        color: playerColor as any,
        indicator: playerIndicator as any,
        showChevron: playerShowChevron,
        bgColor: playerBgColor as any,
      },
      gameMessage: {
        dropCap: dropCap,
        dropCapColor: dropCapColor as any,
      },
      thinking: {
        text: thinkingText,
        style: thinkingStyle as any,
      },
      typography: {
        messages: messageFont as any,
      },
    };
    console.log('[ThemeTestPanel] Current theme config:');
    console.log(JSON.stringify(theme, null, 2));
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

            <Divider label="Corners" labelPosition="center" />
            
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

            <Divider label="Background" labelPosition="center" />
            
            <Select
              label="Animation"
              data={BACKGROUND_ANIMATIONS}
              value={bgAnimation}
              onChange={(v) => v && setBgAnimation(v as BackgroundAnimation)}
            />
            <Select
              label="Tint"
              data={BACKGROUND_TINTS}
              value={bgTint}
              onChange={(v) => v && setBgTint(v as BackgroundTint)}
            />

            <Divider label="Player Messages" labelPosition="center" />
            
            <Select
              label="Player Color"
              data={THEME_COLORS}
              value={playerColor}
              onChange={(v) => v && setPlayerColor(v as ThemeColor)}
            />
            <Select
              label="Player BG Color"
              data={PLAYER_BG_COLORS}
              value={playerBgColor}
              onChange={(v) => v && setPlayerBgColor(v as PlayerBgColor)}
            />
            <Select
              label="Player Indicator"
              data={PLAYER_INDICATORS}
              value={playerIndicator}
              onChange={(v) => v && setPlayerIndicator(v as PlayerIndicator)}
            />
            <Switch
              label="Show Chevron"
              checked={playerShowChevron}
              onChange={(e) => setPlayerShowChevron(e.currentTarget.checked)}
            />

            <Divider label="Game Messages" labelPosition="center" />
            
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

            <Divider label="Typography" labelPosition="center" />
            
            <Select
              label="Message Font"
              data={MESSAGE_FONTS}
              value={messageFont}
              onChange={(v) => v && setMessageFont(v as MessageFont)}
            />

            <Divider />

            <Group grow>
              <Button onClick={applyTheme} color="green">
                Apply Theme
              </Button>
            </Group>
            <Button variant="subtle" onClick={logCurrentTheme} size="xs">
              Log to Console
            </Button>

            <Text size="xs" c="dimmed" ta="center">
              This panel is for testing only.
            </Text>
          </Stack>
        </ScrollArea>
      </Drawer>
    </>
  );
}
