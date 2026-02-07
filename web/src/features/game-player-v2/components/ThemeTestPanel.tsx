/**
 * Theme Test Panel
 * 
 * TEMPORARY: For testing theme presets. Remove when done.
 * Allows switching between presets and overriding animation.
 * Changes are applied immediately.
 */

import { useState, useCallback } from 'react';
import {
  Drawer,
  Stack,
  Select,
  Button,
  Text,
  Divider,
  ActionIcon,
  Tooltip,
} from '@mantine/core';
import { useDisclosure } from '@mantine/hooks';
import { IconPalette } from '@tabler/icons-react';
import type { 
  PartialGameTheme, 
  BackgroundAnimation,
} from '../theme/types';
import { PRESETS } from '../theme/presets';

const BACKGROUND_ANIMATIONS: { value: BackgroundAnimation | 'preset'; label: string }[] = [
  { value: 'preset', label: '(Use Preset Default)' },
  { value: 'none', label: 'None' },
  { value: 'stars', label: '✨ Stars (Space/Sci-Fi)' },
  { value: 'bubbles', label: '🫧 Bubbles (Ocean)' },
  { value: 'fireflies', label: '🪲 Fireflies (Fantasy)' },
  { value: 'snow', label: '❄️ Snow' },
  { value: 'matrix', label: '💻 Matrix (Hacker)' },
  { value: 'embers', label: '🔥 Embers (Fire)' },
  { value: 'hyperspace', label: '🚀 Hyperspace (Sci-Fi)' },
  { value: 'sparkles', label: '✨ Sparkles (Magic)' },
  { value: 'hearts', label: '💕 Hearts (Romance)' },
  { value: 'glitch', label: '⚡ Glitch (Corrupted)' },
  { value: 'circuits', label: '🔌 Circuits (Tech)' },
  { value: 'leaves', label: '🍃 Leaves (Nature)' },
  { value: 'geometric', label: '🔷 Geometric (Abstract)' },
  { value: 'confetti', label: '🎉 Confetti (Playful)' },
];

const PRESET_OPTIONS = [
  { value: '', label: '-- Select Preset --' },
  { value: 'default', label: '⭐ Default (Neutral)' },
  { value: 'minimal', label: 'Minimal / Clean' },
  { value: 'medieval', label: 'Medieval' },
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
  { value: 'fire', label: 'Fire / Ember' },
  { value: 'desert', label: 'Desert / Arid' },
  { value: 'tech', label: '💻 Tech / Digital' },
  { value: 'greenFantasy', label: '🌿 Green Fantasy / Nature Magic' },
  { value: 'abstract', label: '🎨 Abstract / Artistic' },
  { value: 'romance', label: '💕 Romance / Love' },
  { value: 'glitch', label: '⚡ Glitch / Corrupted' },
  { value: 'snowy', label: '❄️ Snowy / Cold' },
];

interface ThemeTestPanelProps {
  currentTheme: PartialGameTheme | undefined;
  onThemeChange: (theme: PartialGameTheme) => void;
}

export function ThemeTestPanel({ onThemeChange }: ThemeTestPanelProps) {
  const [opened, { open, close }] = useDisclosure(false);
  
  const [selectedPreset, setSelectedPreset] = useState<string>('');
  const [animationOverride, setAnimationOverride] = useState<BackgroundAnimation | 'preset'>('preset');

  const applyTheme = useCallback((presetName: string, animation: BackgroundAnimation | 'preset') => {
    const presetDef = PRESETS[presetName];
    if (!presetDef) return;

    // Deep clone preset theme
    const theme: PartialGameTheme = JSON.parse(JSON.stringify(presetDef.theme));

    // Apply animation override
    if (animation !== 'preset') {
      theme.background = { ...theme.background, animation };
    }

    onThemeChange(theme);
  }, [onThemeChange]);

  const handlePresetChange = (value: string | null) => {
    if (!value) return;
    setSelectedPreset(value);
    applyTheme(value, animationOverride);
  };

  const handleAnimationChange = (value: string | null) => {
    if (!value) return;
    const anim = value as BackgroundAnimation | 'preset';
    setAnimationOverride(anim);
    if (selectedPreset) {
      applyTheme(selectedPreset, anim);
    }
  };

  const logCurrentTheme = () => {
    const presetDef = selectedPreset ? PRESETS[selectedPreset] : undefined;
    console.log('[ThemeTestPanel] Preset:', selectedPreset || '(none)');
    console.log('[ThemeTestPanel] Animation override:', animationOverride);
    if (presetDef) {
      console.log('[ThemeTestPanel] Resolved theme:', JSON.stringify(presetDef.theme, null, 2));
    }
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
      >
        <Stack gap="md" px="md">
          <Select
            label="Preset"
            description="Select a theme preset to preview"
            data={PRESET_OPTIONS}
            value={selectedPreset}
            onChange={handlePresetChange}
            placeholder="Select a preset..."
          />

          <Divider label="Overrides" labelPosition="center" />

          <Select
            label="Animation Override"
            description="Override the preset's default background animation"
            data={BACKGROUND_ANIMATIONS}
            value={animationOverride}
            onChange={handleAnimationChange}
          />

          <Divider />

          <Button variant="subtle" onClick={logCurrentTheme} size="xs">
            Log to Console
          </Button>

          <Text size="xs" c="dimmed" ta="center">
            This panel is for testing only. Changes apply immediately.
          </Text>
        </Stack>
      </Drawer>
    </>
  );
}
