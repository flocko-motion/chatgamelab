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

/** Detective / Noir - grounded, stylish, classic */
const detectivePreset: PresetDefinition = {
  theme: {
    corners: { style: 'none', color: 'slate' },
    background: { tint: 'dark' },
    player: { color: 'amber', indicator: 'pipe', indicatorBlink: false, bgColor: 'dark', fontColor: 'light', borderColor: 'amber' },
    gameMessage: { dropCap: false, dropCapColor: 'amber', bgColor: 'black', fontColor: 'light', borderColor: 'slate' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'Investigating...', style: 'dots' },
    typography: { messages: 'serif' },
    statusFields: { bgColor: 'black', accentColor: 'amber', borderColor: 'slate', fontColor: 'light' },
    header: { bgColor: 'black', fontColor: 'light', accentColor: 'amber' },
    divider: { style: 'line', color: 'slate' },
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

export const SIMPLE_PRESETS: Record<string, PresetDefinition> = {
  default: defaultPreset,
  minimal: minimalPreset,
  horror: horrorPreset,
  adventure: adventurePreset,
  detective: detectivePreset,
  barbie: barbiePreset,
  retro: retroPreset,
  western: westernPreset,
  desert: desertPreset,
};
