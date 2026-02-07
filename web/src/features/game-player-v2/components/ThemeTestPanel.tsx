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
  { value: 'school', label: '🏫 School / Educational' },
  { value: 'playful', label: '🎈 Playful / Kids' },
  { value: 'candy', label: '🍬 Candy / Sweet' },
  { value: 'sunshine', label: '☀️ Sunshine / Cheerful' },
  { value: 'storybook', label: '📖 Storybook / Classic' },
  { value: 'barbie', label: '💅 Barbie / Pink Dream' },
  { value: 'superhero', label: '🦸 Superhero / Comic' },
  { value: 'circus', label: '🎪 Circus / Showtime' },
  { value: 'fairy', label: '🧚 Fairy / Enchanted' },
  { value: 'adventure', label: '🗺️ Adventure / Exploration' },
  { value: 'medieval', label: '⚔️ Medieval / Fantasy' },
  { value: 'pirate', label: '🏴‍☠️ Pirate / Nautical' },
  { value: 'western', label: '🤠 Western / Wild West' },
  { value: 'steampunk', label: '⚙️ Steampunk / Victorian' },
  { value: 'greenFantasy', label: '🌿 Green Fantasy / Nature Magic' },
  { value: 'mystic', label: '🔮 Mystic / Occult' },
  { value: 'nature', label: '🌲 Nature / Forest' },
  { value: 'jungle', label: '🌴 Jungle / Tropical' },
  { value: 'garden', label: '🌷 Garden / Flowers' },
  { value: 'ocean', label: '🌊 Ocean / Coastal' },
  { value: 'underwater', label: '🐠 Underwater / Deep Sea' },
  { value: 'desert', label: '🏜️ Desert / Arid' },
  { value: 'snowy', label: '❄️ Snowy / Winter' },
  { value: 'fire', label: '🔥 Fire / Volcanic' },
  { value: 'horror', label: '👻 Horror / Dark' },
  { value: 'mystery', label: '🔍 Mystery / Whodunit' },
  { value: 'detective', label: '🕵️ Detective / Classic' },
  { value: 'noir', label: '🖤 Noir / Moody' },
  { value: 'zombie', label: '🧟 Zombie / Apocalypse' },
  { value: 'scifi', label: '🚀 Sci-Fi / Futuristic' },
  { value: 'cyberpunk', label: '💜 Cyberpunk / Neon' },
  { value: 'space', label: '🌌 Space / Cosmic' },
  { value: 'tech', label: '💻 Tech / Digital' },
  { value: 'terminal', label: '💚 Terminal (Green/Black)' },
  { value: 'hacker', label: '🔴 Hacker (Red/Green)' },
  { value: 'glitch', label: '⚡ Glitch / Corrupted' },
  { value: 'retro', label: '📼 Retro / 80s' },
  { value: 'romance', label: '💕 Romance / Love' },
  { value: 'abstract', label: '🎨 Abstract / Artistic' },
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
