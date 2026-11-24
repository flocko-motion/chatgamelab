import { useGenres } from "@/api/hooks";
import type { GameField, VoiceSlot } from "@/api/types";

/**
 * Whether the editor shows a game field for a genre: the engine says which
 * fields each genre reads. Until the genres are known, and for a genre the
 * engine does not list, every field shows, so nothing a game holds is hidden.
 */
export function useGenreFields(genre: string): (field: GameField) => boolean {
  const { data: genres } = useGenres();
  const fields = genres?.find((g) => g.id === genre)?.fields;
  return (field) => !fields || fields.includes(field);
}

/**
 * The voices a genre speaks in and what a game may choose for each, as the
 * engine says; none until the genres are known.
 */
export function useVoiceSlots(genre: string): VoiceSlot[] {
  const { data: genres } = useGenres();
  return genres?.find((g) => g.id === genre)?.voiceSlots ?? [];
}

/** The choice a game made for one of its voices; one it chose nothing for is any. */
export function voiceChoice(voices: Record<string, string>, slot: VoiceSlot): string {
  return voices[slot.name] ?? "any";
}

/** A game's voices with one of them set to a choice. */
export function withVoice(voices: Record<string, string>, slot: VoiceSlot, choice: string): Record<string, string> {
  return { ...voices, [slot.name]: choice };
}

/** Whether a game's voices set one of them to a voice of its own. */
export function hasCustomVoice(voices: Record<string, string>, slots: VoiceSlot[]): boolean {
  return slots.some((slot) => voices[slot.name] === "custom");
}
