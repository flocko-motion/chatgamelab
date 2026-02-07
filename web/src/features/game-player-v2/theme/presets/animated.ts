/**
 * Animated Presets (with background animation)
 *
 * Theme configurations that include particle/animation effects.
 */

import type { PresetDefinition } from './types';

/** Sci-fi / Cyberpunk */
const scifiPreset: PresetDefinition = {
  theme: {
    corners: { style: 'brackets', color: 'cyan' },
    background: { tint: 'black', animation: 'stars' },
    player: { color: 'cyan', indicator: 'cursor', indicatorBlink: true, bgColor: 'cyan', fontColor: 'light', borderColor: 'cyan' },
    gameMessage: { dropCap: false, dropCapColor: 'cyan', bgColor: 'dark', fontColor: 'cyan', borderColor: 'cyan' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'Processing...', style: 'dots' },
    typography: { messages: 'mono' },
    statusFields: { bgColor: 'dark', accentColor: 'cyan', borderColor: 'cyan', fontColor: 'cyan' },
    header: { bgColor: 'black', fontColor: 'cyan', accentColor: 'cyan' },
    divider: { style: 'line', color: 'cyan' },
  },
};

/** Medieval */
const medievalPreset: PresetDefinition = {
  theme: {
    corners: { style: 'flourish', color: 'amber' },
    background: { tint: 'warm', animation: 'fireflies' },
    player: { color: 'amber', indicator: 'dot', indicatorBlink: false, bgColor: 'creme', fontColor: 'dark', borderColor: 'amber' },
    gameMessage: { dropCap: true, dropCapColor: 'amber', bgColor: 'creme', fontColor: 'dark', borderColor: 'amber' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'The tale continues...', style: 'typewriter' },
    typography: { messages: 'serif' },
    statusFields: { bgColor: 'creme', accentColor: 'amber', borderColor: 'amber', fontColor: 'dark' },
    header: { bgColor: 'creme', fontColor: 'dark', accentColor: 'amber' },
    divider: { style: 'diamond', color: 'amber' },
  },
};

/** Mystery / Mystic - purple, magical, ethereal */
const mysteryPreset: PresetDefinition = {
  theme: {
    corners: { style: 'dots', color: 'violet' },
    background: { tint: 'darkViolet', animation: 'fireflies' },
    player: { color: 'violet', indicator: 'star', indicatorBlink: true, bgColor: 'violet', fontColor: 'light', borderColor: 'violet' },
    gameMessage: { dropCap: true, dropCapColor: 'violet', bgColor: 'dark', fontColor: 'violet', borderColor: 'violet' },
    cards: { borderThickness: 'medium' },
    thinking: { text: 'The veil thins...', style: 'pulse' },
    typography: { messages: 'serif' },
    statusFields: { bgColor: 'dark', accentColor: 'violet', borderColor: 'violet', fontColor: 'violet' },
    header: { bgColor: 'dark', fontColor: 'violet', accentColor: 'violet' },
    divider: { style: 'star', color: 'violet' },
  },
};

/** Space / Cosmic */
const spacePreset: PresetDefinition = {
  theme: {
    corners: { style: 'brackets', color: 'cyan' },
    background: { tint: 'dark', animation: 'hyperspace' },
    player: { color: 'cyan', indicator: 'dot', indicatorBlink: true, bgColor: 'dark', fontColor: 'light', borderColor: 'cyan' },
    gameMessage: { dropCap: false, dropCapColor: 'cyan', bgColor: 'dark', fontColor: 'light', borderColor: 'cyan' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'Scanning...', style: 'spinner' },
    typography: { messages: 'mono' },
    statusFields: { bgColor: 'dark', accentColor: 'cyan', borderColor: 'cyan', fontColor: 'light' },
    header: { bgColor: 'dark', fontColor: 'light', accentColor: 'cyan' },
    divider: { style: 'star', color: 'cyan' },
  },
};

/** Terminal - Green on black, classic */
const terminalPreset: PresetDefinition = {
  theme: {
    corners: { style: 'brackets', color: 'hacker' },
    background: { tint: 'black', animation: 'matrix' },
    player: { color: 'hacker', indicator: 'underscore', indicatorBlink: true, bgColor: 'black', fontColor: 'hacker', borderColor: 'hacker' },
    gameMessage: { dropCap: false, dropCapColor: 'hacker', bgColor: 'black', fontColor: 'hacker', borderColor: 'hacker' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'Loading...', style: 'dots', streamingCursor: 'pipe' },
    typography: { messages: 'mono' },
    statusFields: { bgColor: 'black', accentColor: 'hacker', borderColor: 'hacker', fontColor: 'hacker' },
    header: { bgColor: 'black', fontColor: 'hacker', accentColor: 'hacker' },
    divider: { style: 'dash', color: 'hacker' },
  },
};

/** Hacker - Aggressive (Red AI, Green User) */
const hackerPreset: PresetDefinition = {
  theme: {
    corners: { style: 'brackets', color: 'terminal' },
    background: { tint: 'black', animation: 'matrix' },
    player: { color: 'hacker', indicator: 'underscore', indicatorBlink: true, bgColor: 'green', fontColor: 'hacker', borderColor: 'hacker' },
    gameMessage: { dropCap: false, dropCapColor: 'terminal', bgColor: 'red', fontColor: 'terminal', borderColor: 'terminal' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'EXECUTING...', style: 'dots', streamingCursor: 'pipe' },
    typography: { messages: 'mono' },
    statusFields: { bgColor: 'black', accentColor: 'terminal', borderColor: 'terminal', fontColor: 'terminal' },
    header: { bgColor: 'black', fontColor: 'terminal', accentColor: 'terminal' },
    divider: { style: 'dash', color: 'terminal' },
  },
};

/** Playful / Kids - Rainbow colorful theme */
const playfulPreset: PresetDefinition = {
  theme: {
    corners: { style: 'dots', color: 'orange' },
    background: { tint: 'blue', animation: 'confetti' },
    player: { color: 'orange', indicator: 'star', indicatorBlink: true, bgColor: 'orangeLight', fontColor: 'dark', borderColor: 'orange' },
    gameMessage: { dropCap: true, dropCapColor: 'violet', bgColor: 'violetLight', fontColor: 'dark', borderColor: 'violet' },
    cards: { borderThickness: 'thick' },
    thinking: { text: 'Magic is happening...', style: 'pulse' },
    typography: { messages: 'sans' },
    statusFields: { bgColor: 'cyanLight', accentColor: 'cyan', borderColor: 'cyan', fontColor: 'dark' },
    header: { bgColor: 'greenLight', fontColor: 'dark', accentColor: 'emerald' },
    divider: { style: 'star', color: 'pink' },
  },
};

/** Nature / Forest */
const naturePreset: PresetDefinition = {
  theme: {
    corners: { style: 'flourish', color: 'emerald' },
    background: { tint: 'warm', animation: 'leaves' },
    player: { color: 'emerald', indicator: 'chevron', indicatorBlink: false, bgColor: 'creme', fontColor: 'dark', borderColor: 'emerald' },
    gameMessage: { dropCap: true, dropCapColor: 'emerald', bgColor: 'creme', fontColor: 'dark', borderColor: 'emerald' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'The forest whispers...', style: 'typewriter' },
    typography: { messages: 'serif' },
    statusFields: { bgColor: 'creme', accentColor: 'emerald', borderColor: 'emerald', fontColor: 'dark' },
    header: { bgColor: 'creme', fontColor: 'dark', accentColor: 'emerald' },
    divider: { style: 'dot', color: 'emerald' },
  },
};

/** Ocean / Underwater */
const oceanPreset: PresetDefinition = {
  theme: {
    corners: { style: 'arrows', color: 'cyan' },
    background: { tint: 'cool', animation: 'bubbles' },
    player: { color: 'cyan', indicator: 'dot', indicatorBlink: false, bgColor: 'cyanLight', fontColor: 'dark', borderColor: 'cyan' },
    gameMessage: { dropCap: true, dropCapColor: 'cyan', bgColor: 'blueLight', fontColor: 'dark', borderColor: 'cyan' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'Bubbles rise...', style: 'dots' },
    typography: { messages: 'sans' },
    statusFields: { bgColor: 'blueLight', accentColor: 'cyan', borderColor: 'cyan', fontColor: 'dark' },
    header: { bgColor: 'blueLight', fontColor: 'dark', accentColor: 'cyan' },
    divider: { style: 'dots', color: 'cyan' },
  },
};

/** Fire / Ember - Destruction, firefighting, volcanic */
const firePreset: PresetDefinition = {
  theme: {
    corners: { style: 'none', color: 'orange' },
    background: { tint: 'dark', animation: 'embers' },
    player: { color: 'orange', indicator: 'chevron', indicatorBlink: false, bgColor: 'dark', fontColor: 'light', borderColor: 'orange' },
    gameMessage: { dropCap: false, dropCapColor: 'orange', bgColor: 'dark', fontColor: 'light', borderColor: 'orange' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'Flames crackle...', style: 'pulse' },
    typography: { messages: 'sans' },
    statusFields: { bgColor: 'dark', accentColor: 'orange', borderColor: 'orange', fontColor: 'light' },
    header: { bgColor: 'dark', fontColor: 'light', accentColor: 'orange' },
    divider: { style: 'line', color: 'orange' },
  },
};

/** Tech - Modern technology, clean digital, shiny */
const techPreset: PresetDefinition = {
  theme: {
    corners: { style: 'brackets', color: 'cyan' },
    background: { tint: 'cool', animation: 'circuits' },
    player: { color: 'cyan', indicator: 'cursor', indicatorBlink: true, bgColor: 'cyanLight', fontColor: 'dark', borderColor: 'cyan' },
    gameMessage: { dropCap: false, dropCapColor: 'cyan', bgColor: 'white', fontColor: 'dark', borderColor: 'cyan' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'Processing...', style: 'spinner' },
    typography: { messages: 'mono' },
    statusFields: { bgColor: 'cyanLight', accentColor: 'cyan', borderColor: 'cyan', fontColor: 'dark' },
    header: { bgColor: 'white', fontColor: 'dark', accentColor: 'cyan' },
    divider: { style: 'line', color: 'cyan' },
  },
};

/** Green Fantasy - Enchanted forest, nature magic */
const greenFantasyPreset: PresetDefinition = {
  theme: {
    corners: { style: 'flourish', color: 'emerald' },
    background: { tint: 'green', animation: 'sparkles' },
    player: { color: 'emerald', indicator: 'star', indicatorBlink: true, bgColor: 'greenLight', fontColor: 'dark', borderColor: 'emerald' },
    gameMessage: { dropCap: true, dropCapColor: 'emerald', bgColor: 'greenLight', fontColor: 'dark', borderColor: 'emerald' },
    cards: { borderThickness: 'medium' },
    thinking: { text: 'Magic awakens...', style: 'pulse' },
    typography: { messages: 'fantasy' },
    statusFields: { bgColor: 'greenLight', accentColor: 'emerald', borderColor: 'emerald', fontColor: 'dark' },
    header: { bgColor: 'greenLight', fontColor: 'dark', accentColor: 'emerald' },
    divider: { style: 'star', color: 'emerald' },
  },
};

/** Abstract - Artistic, geometric, creative */
const abstractPreset: PresetDefinition = {
  theme: {
    corners: { style: 'dots', color: 'violet' },
    background: { tint: 'darkViolet', animation: 'geometric' },
    player: { color: 'cyan', indicator: 'diamond', indicatorBlink: false, bgColor: 'violet', fontColor: 'light', borderColor: 'cyan' },
    gameMessage: { dropCap: false, dropCapColor: 'violet', bgColor: 'dark', fontColor: 'light', borderColor: 'violet' },
    cards: { borderThickness: 'thick' },
    thinking: { text: 'Creating...', style: 'spinner' },
    typography: { messages: 'sans' },
    statusFields: { bgColor: 'dark', accentColor: 'violet', borderColor: 'violet', fontColor: 'light' },
    header: { bgColor: 'violet', fontColor: 'light', accentColor: 'cyan' },
    divider: { style: 'diamond', color: 'violet' },
  },
};

/** Romance - Soft, warm, romantic */
const romancePreset: PresetDefinition = {
  theme: {
    corners: { style: 'flourish', color: 'rose' },
    background: { tint: 'pink', animation: 'hearts' },
    player: { color: 'rose', indicator: 'diamond', indicatorBlink: false, bgColor: 'roseLight', fontColor: 'dark', borderColor: 'rose' },
    gameMessage: { dropCap: true, dropCapColor: 'rose', bgColor: 'roseLight', fontColor: 'dark', borderColor: 'rose' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'Hearts flutter...', style: 'pulse' },
    typography: { messages: 'serif' },
    statusFields: { bgColor: 'roseLight', accentColor: 'rose', borderColor: 'rose', fontColor: 'dark' },
    header: { bgColor: 'roseLight', fontColor: 'dark', accentColor: 'rose' },
    divider: { style: 'diamond', color: 'rose' },
  },
};

/** Glitch - Corrupted, digital chaos, cyberpunk error */
const glitchPreset: PresetDefinition = {
  theme: {
    corners: { style: 'cursor', color: 'terminal', blink: true },
    background: { tint: 'black', animation: 'glitch' },
    player: { color: 'cyan', indicator: 'underscore', indicatorBlink: true, bgColor: 'dark', fontColor: 'cyan', borderColor: 'terminal' },
    gameMessage: { dropCap: false, dropCapColor: 'terminal', bgColor: 'black', fontColor: 'terminal', borderColor: 'hacker' },
    cards: { borderThickness: 'medium' },
    thinking: { text: 'ERR0R...', style: 'dots', streamingCursor: 'block' },
    typography: { messages: 'mono' },
    statusFields: { bgColor: 'black', accentColor: 'terminal', borderColor: 'hacker', fontColor: 'terminal' },
    header: { bgColor: 'black', fontColor: 'terminal', accentColor: 'hacker' },
    divider: { style: 'dash', color: 'terminal' },
  },
};

/** Snowy / Cold - Winter wonderland */
const snowyPreset: PresetDefinition = {
  theme: {
    corners: { style: 'flourish', color: 'slate' },
    background: { tint: 'cool', animation: 'snow' },
    player: { color: 'cyan', indicator: 'diamond', indicatorBlink: false, bgColor: 'cyanLight', fontColor: 'dark', borderColor: 'cyan' },
    gameMessage: { dropCap: true, dropCapColor: 'cyan', bgColor: 'white', fontColor: 'dark', borderColor: 'slate' },
    cards: { borderThickness: 'thin' },
    thinking: { text: 'Snowflakes falling...', style: 'dots' },
    typography: { messages: 'serif' },
    statusFields: { bgColor: 'white', accentColor: 'cyan', borderColor: 'slate', fontColor: 'dark' },
    header: { bgColor: 'white', fontColor: 'dark', accentColor: 'cyan' },
    divider: { style: 'star', color: 'slate' },
  },
};

export const ANIMATED_PRESETS: Record<string, PresetDefinition> = {
  scifi: scifiPreset,
  medieval: medievalPreset,
  mystery: mysteryPreset,
  space: spacePreset,
  terminal: terminalPreset,
  hacker: hackerPreset,
  playful: playfulPreset,
  nature: naturePreset,
  ocean: oceanPreset,
  fire: firePreset,
  tech: techPreset,
  greenFantasy: greenFantasyPreset,
  abstract: abstractPreset,
  romance: romancePreset,
  glitch: glitchPreset,
  snowy: snowyPreset,
};
