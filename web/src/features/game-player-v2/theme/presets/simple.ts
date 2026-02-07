/**
 * Simple Presets (no background animation)
 *
 * Static theme configurations without particle effects.
 */

import type { PresetDefinition } from './types';

/** Default / Neutral - clean, minimal */
const defaultPreset: PresetDefinition = {
  theme: {
    corners: { style: 'none', color: 'slate' },
    background: { tint: 'neutral' },
    player: { color: 'slate', indicator: 'chevron', indicatorBlink: false, bgColor: 'white', fontColor: 'dark', borderColor: 'slate' },
    gameMessage: { dropCap: false, dropCapColor: 'slate', bgColor: 'white', fontColor: 'dark', borderColor: 'slate' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'The story unfolds...', style: 'dots' },
    typography: { messages: 'sans' },
    statusFields: { bgColor: 'white', accentColor: 'slate', borderColor: 'slate', fontColor: 'dark' },
    header: { bgColor: 'white', fontColor: 'dark', accentColor: 'slate' },
    divider: { style: 'none', color: 'slate' },
  },
};

/** Clean / Minimal */
const minimalPreset: PresetDefinition = {
  theme: {
    corners: { style: 'none', color: 'slate' },
    background: { tint: 'neutral' },
    player: { color: 'slate', indicator: 'none', indicatorBlink: false, bgColor: 'white', fontColor: 'dark', borderColor: 'slate' },
    gameMessage: { dropCap: false, dropCapColor: 'slate', bgColor: 'white', fontColor: 'dark', borderColor: 'slate' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'Loading...', style: 'dots' },
    typography: { messages: 'sans' },
    statusFields: { bgColor: 'white', accentColor: 'slate', borderColor: 'slate', fontColor: 'dark' },
    header: { bgColor: 'white', fontColor: 'dark', accentColor: 'slate' },
    divider: { style: 'line', color: 'slate' },
  },
};

/** Horror / Mystery */
const horrorPreset: PresetDefinition = {
  theme: {
    corners: { style: 'none', color: 'slate' },
    background: { tint: 'dark' },
    player: { color: 'rose', indicator: 'none', indicatorBlink: false, bgColor: 'dark', fontColor: 'light', borderColor: 'rose' },
    gameMessage: { dropCap: false, dropCapColor: 'rose', bgColor: 'dark', fontColor: 'light', borderColor: 'slate' },
    cards: { borderThickness: 'none' },
    thinking: { text: 'Something stirs...', style: 'pulse' },
    typography: { messages: 'serif' },
    statusFields: { bgColor: 'dark', accentColor: 'rose', borderColor: 'slate', fontColor: 'light' },
    header: { bgColor: 'dark', fontColor: 'light', accentColor: 'rose' },
    divider: { style: 'none', color: 'slate' },
  },
};

/** Adventure / Exploration */
const adventurePreset: PresetDefinition = {
  theme: {
    corners: { style: 'arrows', color: 'emerald' },
    background: { tint: 'neutral' },
    player: { color: 'emerald', indicator: 'chevron', indicatorBlink: false, bgColor: 'creme', fontColor: 'dark', borderColor: 'emerald' },
    gameMessage: { dropCap: false, dropCapColor: 'emerald', bgColor: 'white', fontColor: 'dark', borderColor: 'emerald' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'The journey continues...', style: 'dots' },
    typography: { messages: 'sans' },
    statusFields: { bgColor: 'creme', accentColor: 'emerald', borderColor: 'emerald', fontColor: 'dark' },
    header: { bgColor: 'white', fontColor: 'dark', accentColor: 'emerald' },
    divider: { style: 'dot', color: 'emerald' },
  },
};

/** Detective - Classic whodunit, warm amber tones, magnifying glass feel */
const detectivePreset: PresetDefinition = {
  theme: {
    corners: { style: 'none', color: 'amber' },
    background: { tint: 'warm' },
    player: { color: 'amber', indicator: 'pipe', indicatorBlink: false, bgColor: 'creme', fontColor: 'dark', borderColor: 'amber' },
    gameMessage: { dropCap: false, dropCapColor: 'amber', bgColor: 'creme', fontColor: 'dark', borderColor: 'amber' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'Investigating...', style: 'dots' },
    typography: { messages: 'serif' },
    statusFields: { bgColor: 'creme', accentColor: 'amber', borderColor: 'amber', fontColor: 'dark' },
    header: { bgColor: 'creme', fontColor: 'dark', accentColor: 'amber' },
    divider: { style: 'line', color: 'amber' },
  },
};

/** Noir - Dark, moody, black & white with stark contrast */
const noirPreset: PresetDefinition = {
  theme: {
    corners: { style: 'none', color: 'slate' },
    background: { tint: 'black' },
    player: { color: 'slate', indicator: 'none', indicatorBlink: false, bgColor: 'black', fontColor: 'light', borderColor: 'slate' },
    gameMessage: { dropCap: false, dropCapColor: 'slate', bgColor: 'black', fontColor: 'light', borderColor: 'slate' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'The shadows speak...', style: 'pulse' },
    typography: { messages: 'serif' },
    statusFields: { bgColor: 'black', accentColor: 'slate', borderColor: 'slate', fontColor: 'light' },
    header: { bgColor: 'black', fontColor: 'light', accentColor: 'slate' },
    divider: { style: 'line', color: 'slate' },
  },
};

/** Steampunk - Brass, gears, Victorian industrial */
const steampunkPreset: PresetDefinition = {
  theme: {
    corners: { style: 'flourish', color: 'brown' },
    background: { tint: 'warm' },
    player: { color: 'brown', indicator: 'dot', indicatorBlink: false, bgColor: 'creme', fontColor: 'dark', borderColor: 'brown' },
    gameMessage: { dropCap: true, dropCapColor: 'amber', bgColor: 'creme', fontColor: 'dark', borderColor: 'brown' },
    cards: { borderThickness: 'medium' },
    thinking: { text: 'Gears turning...', style: 'spinner' },
    typography: { messages: 'serif' },
    statusFields: { bgColor: 'creme', accentColor: 'amber', borderColor: 'brown', fontColor: 'dark' },
    header: { bgColor: 'creme', fontColor: 'dark', accentColor: 'brown' },
    divider: { style: 'diamond', color: 'brown' },
  },
};

/** Zombie - Post-apocalyptic, decayed, eerie green */
const zombiePreset: PresetDefinition = {
  theme: {
    corners: { style: 'none', color: 'emerald' },
    background: { tint: 'dark' },
    player: { color: 'emerald', indicator: 'none', indicatorBlink: false, bgColor: 'dark', fontColor: 'light', borderColor: 'emerald' },
    gameMessage: { dropCap: false, dropCapColor: 'emerald', bgColor: 'dark', fontColor: 'light', borderColor: 'emerald' },
    cards: { borderThickness: 'none' },
    thinking: { text: 'They are coming...', style: 'pulse' },
    typography: { messages: 'sans' },
    statusFields: { bgColor: 'dark', accentColor: 'emerald', borderColor: 'emerald', fontColor: 'light' },
    header: { bgColor: 'dark', fontColor: 'light', accentColor: 'emerald' },
    divider: { style: 'dash', color: 'emerald' },
  },
};

/** Barbie / Pink Dream */
const barbiePreset: PresetDefinition = {
  theme: {
    corners: { style: 'flourish', color: 'pink' },
    background: { tint: 'pink' },
    player: { color: 'pink', indicator: 'diamond', indicatorBlink: false, bgColor: 'pinkLight', fontColor: 'pink', borderColor: 'pink' },
    gameMessage: { dropCap: true, dropCapColor: 'pink', bgColor: 'pinkLight', fontColor: 'dark', borderColor: 'pink' },
    cards: { borderThickness: 'medium' },
    thinking: { text: 'Fabulous things await...', style: 'pulse' },
    typography: { messages: 'serif' },
    statusFields: { bgColor: 'pinkLight', accentColor: 'pink', borderColor: 'pink', fontColor: 'pink' },
    header: { bgColor: 'pinkLight', fontColor: 'pink', accentColor: 'pink' },
    divider: { style: 'diamond', color: 'pink' },
  },
};

/** Retro / 80s */
const retroPreset: PresetDefinition = {
  theme: {
    corners: { style: 'brackets', color: 'violet' },
    background: { tint: 'dark' },
    player: { color: 'cyan', indicator: 'cursor', indicatorBlink: true, bgColor: 'dark', fontColor: 'light', borderColor: 'cyan' },
    gameMessage: { dropCap: false, dropCapColor: 'violet', bgColor: 'dark', fontColor: 'light', borderColor: 'violet' },
    cards: { borderThickness: 'medium' },
    thinking: { text: 'Loading...', style: 'spinner' },
    typography: { messages: 'mono' },
    statusFields: { bgColor: 'dark', accentColor: 'violet', borderColor: 'violet', fontColor: 'light' },
    header: { bgColor: 'dark', fontColor: 'light', accentColor: 'violet' },
    divider: { style: 'line', color: 'violet' },
  },
};

/** Western / Wild West */
const westernPreset: PresetDefinition = {
  theme: {
    corners: { style: 'arrows', color: 'amber' },
    background: { tint: 'warm' },
    player: { color: 'amber', indicator: 'dot', indicatorBlink: false, bgColor: 'creme', fontColor: 'dark', borderColor: 'amber' },
    gameMessage: { dropCap: true, dropCapColor: 'amber', bgColor: 'creme', fontColor: 'dark', borderColor: 'amber' },
    cards: { borderThickness: 'medium' },
    thinking: { text: 'Dust settles...', style: 'typewriter' },
    typography: { messages: 'serif' },
    statusFields: { bgColor: 'creme', accentColor: 'amber', borderColor: 'amber', fontColor: 'dark' },
    header: { bgColor: 'creme', fontColor: 'dark', accentColor: 'amber' },
    divider: { style: 'star', color: 'amber' },
  },
};

/** Desert - Arid, sandy, hot climate */
const desertPreset: PresetDefinition = {
  theme: {
    corners: { style: 'arrows', color: 'amber' },
    background: { tint: 'warm' },
    player: { color: 'amber', indicator: 'chevron', indicatorBlink: false, bgColor: 'amberLight', fontColor: 'dark', borderColor: 'amber' },
    gameMessage: { dropCap: false, dropCapColor: 'amber', bgColor: 'creme', fontColor: 'dark', borderColor: 'amber' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'Heat shimmers...', style: 'dots' },
    typography: { messages: 'sans' },
    statusFields: { bgColor: 'amberLight', accentColor: 'amber', borderColor: 'amber', fontColor: 'dark' },
    header: { bgColor: 'amberLight', fontColor: 'dark', accentColor: 'amber' },
    divider: { style: 'dot', color: 'amber' },
  },
};

/** School - Friendly, educational, clean blue/white */
const schoolPreset: PresetDefinition = {
  theme: {
    corners: { style: 'none', color: 'sky' },
    background: { tint: 'neutral' },
    player: { color: 'sky', indicator: 'chevron', indicatorBlink: false, bgColor: 'skyLight', fontColor: 'dark', borderColor: 'sky' },
    gameMessage: { dropCap: false, dropCapColor: 'sky', bgColor: 'white', fontColor: 'dark', borderColor: 'sky' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'Thinking...', style: 'dots' },
    typography: { messages: 'sans' },
    statusFields: { bgColor: 'skyLight', accentColor: 'sky', borderColor: 'sky', fontColor: 'dark' },
    header: { bgColor: 'skyLight', fontColor: 'dark', accentColor: 'sky' },
    divider: { style: 'line', color: 'sky' },
  },
};

/** Candy - Sweet, colorful, pastel rainbow */
const candyPreset: PresetDefinition = {
  theme: {
    corners: { style: 'dots', color: 'coral' },
    background: { tint: 'pink' },
    player: { color: 'coral', indicator: 'star', indicatorBlink: false, bgColor: 'coralLight', fontColor: 'dark', borderColor: 'coral' },
    gameMessage: { dropCap: true, dropCapColor: 'lavender', bgColor: 'lavenderLight', fontColor: 'dark', borderColor: 'lavender' },
    cards: { borderThickness: 'medium' },
    thinking: { text: 'Sweet things coming...', style: 'pulse' },
    typography: { messages: 'sans' },
    statusFields: { bgColor: 'sunshineLight', accentColor: 'sunshine', borderColor: 'coral', fontColor: 'dark' },
    header: { bgColor: 'coralLight', fontColor: 'dark', accentColor: 'coral' },
    divider: { style: 'star', color: 'coral' },
  },
};

/** Superhero - Bold, bright, comic book feel */
const superheroPreset: PresetDefinition = {
  theme: {
    corners: { style: 'arrows', color: 'indigo' },
    background: { tint: 'blue' },
    player: { color: 'coral', indicator: 'chevron', indicatorBlink: false, bgColor: 'coralLight', fontColor: 'dark', borderColor: 'coral' },
    gameMessage: { dropCap: true, dropCapColor: 'indigo', bgColor: 'indigoLight', fontColor: 'dark', borderColor: 'indigo' },
    cards: { borderThickness: 'thick' },
    thinking: { text: 'Hero incoming...', style: 'spinner' },
    typography: { messages: 'sans' },
    statusFields: { bgColor: 'sunshineLight', accentColor: 'sunshine', borderColor: 'indigo', fontColor: 'dark' },
    header: { bgColor: 'indigoLight', fontColor: 'dark', accentColor: 'indigo' },
    divider: { style: 'star', color: 'indigo' },
  },
};

/** Sunshine - Warm, bright, cheerful yellow */
const sunshinePreset: PresetDefinition = {
  theme: {
    corners: { style: 'dots', color: 'sunshine' },
    background: { tint: 'warm' },
    player: { color: 'sunshine', indicator: 'star', indicatorBlink: false, bgColor: 'sunshineLight', fontColor: 'dark', borderColor: 'sunshine' },
    gameMessage: { dropCap: false, dropCapColor: 'sunshine', bgColor: 'sunshineLight', fontColor: 'dark', borderColor: 'sunshine' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'A bright idea forms...', style: 'dots' },
    typography: { messages: 'sans' },
    statusFields: { bgColor: 'sunshineLight', accentColor: 'sunshine', borderColor: 'sunshine', fontColor: 'dark' },
    header: { bgColor: 'sunshineLight', fontColor: 'dark', accentColor: 'sunshine' },
    divider: { style: 'dot', color: 'sunshine' },
  },
};

/** Storybook - Classic children's book, warm, illustrated feel */
const storybookPreset: PresetDefinition = {
  theme: {
    corners: { style: 'flourish', color: 'teal' },
    background: { tint: 'warm' },
    player: { color: 'teal', indicator: 'dot', indicatorBlink: false, bgColor: 'creme', fontColor: 'dark', borderColor: 'teal' },
    gameMessage: { dropCap: true, dropCapColor: 'coral', bgColor: 'creme', fontColor: 'dark', borderColor: 'teal' },
    cards: { borderThickness: 'medium' },
    thinking: { text: 'Once upon a time...', style: 'typewriter' },
    typography: { messages: 'serif' },
    statusFields: { bgColor: 'creme', accentColor: 'teal', borderColor: 'teal', fontColor: 'dark' },
    header: { bgColor: 'creme', fontColor: 'dark', accentColor: 'teal' },
    divider: { style: 'diamond', color: 'teal' },
  },
};

export const SIMPLE_PRESETS: Record<string, PresetDefinition> = {
  default: defaultPreset,
  minimal: minimalPreset,
  horror: horrorPreset,
  adventure: adventurePreset,
  detective: detectivePreset,
  noir: noirPreset,
  steampunk: steampunkPreset,
  zombie: zombiePreset,
  barbie: barbiePreset,
  retro: retroPreset,
  western: westernPreset,
  desert: desertPreset,
  school: schoolPreset,
  candy: candyPreset,
  superhero: superheroPreset,
  sunshine: sunshinePreset,
  storybook: storybookPreset,
};
