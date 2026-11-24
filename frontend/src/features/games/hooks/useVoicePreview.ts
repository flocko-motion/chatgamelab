import { useCallback, useEffect, useRef, useState } from "react";
import { useGenres, useNewVoice, useVoiceSample, type VoiceSample } from "@/api/hooks";
import { extractRawErrorCode } from "@/common/types/errorCodes";
import { genreWording } from "../lib/editorPages";

/** A sample as played: for which genre and voice style, and where to play it from. */
interface Heard {
  key: string;
  url: string;
}

/** Why a sample could not be heard, as a translation key. */
function errorKey(error: unknown): string {
  switch (extractRawErrorCode(error)) {
    case "no_api_key":
      return "games.editor.sound.errors.noKey";
    case "invalid_api_key":
      return "games.editor.sound.errors.keyRejected";
  }
  return "games.editor.sound.errors.failed";
}

/**
 * Hearing a game's voice in the editor: the sample the voice of the genre
 * and voice style says, played here (the overview's tile and the Sound page
 * share one), and making a new voice for the style. A sample heard once
 * plays again without asking the server, until the style changes. Whether a
 * new voice can be made at all is the engine's to say (the genre's
 * makesVoices); without a voice style the built-in voice speaks.
 */
export function useVoicePreview(genre: string, voiceStyle: string, scenario: string) {
  const { data: genres } = useGenres();
  const makesVoices = !!genres?.find((g) => g.id === genre)?.makesVoices;
  const style = voiceStyle.trim();
  const key = `${genre}\n${style}`;

  const sample = useVoiceSample();
  const renewal = useNewVoice();
  const [heard, setHeard] = useState<Heard | null>(null);
  const [playing, setPlaying] = useState(false);
  const [failed, setFailed] = useState<string | null>(null);
  const audio = useRef<HTMLAudioElement | null>(null);

  const stop = useCallback(() => {
    audio.current?.pause();
    audio.current = null;
    setPlaying(false);
  }, []);

  const start = (url: string) => {
    stop();
    const player = new Audio(url);
    player.onended = () => setPlaying(false);
    audio.current = player;
    setPlaying(true);
    player.play().catch(() => setPlaying(false));
  };

  const keep = (got: VoiceSample) => {
    if (heard) URL.revokeObjectURL(heard.url);
    const next = { key, url: URL.createObjectURL(got.audio) };
    setHeard(next);
    return next;
  };

  /** Plays the sample, or stops it while it plays. */
  const play = async () => {
    if (playing) return stop();
    setFailed(null);
    if (heard?.key === key) return start(heard.url);
    try {
      start(keep(await sample.mutateAsync({ genre, voiceStyle: style, scenario })).url);
    } catch (error) {
      setFailed(errorKey(error));
    }
  };

  /** Replaces the voice for the style on the paying key, and plays the new one. */
  const renew = async () => {
    stop();
    setFailed(null);
    try {
      start(keep(await renewal.mutateAsync({ genre, voiceStyle: style, scenario })).url);
    } catch (error) {
      setFailed(errorKey(error));
    }
  };

  // The sample stops, and its audio goes, when the editor closes.
  useEffect(
    () => () => {
      audio.current?.pause();
    },
    [],
  );
  useEffect(
    () => () => {
      if (heard) URL.revokeObjectURL(heard.url);
    },
    [heard],
  );

  return {
    /** The genre makes a voice of its own per key and style, which can be remade. */
    makesVoices,
    /** The game has a voice style; without one the built-in voice speaks. */
    hasStyle: style !== "",
    /** A new voice can be made: the genre makes them, and there is a style. */
    mayRenew: makesVoices && style !== "",
    playing,
    /** The sample is being fetched or made. */
    loading: sample.isPending,
    /** A new voice is being made. */
    renewing: renewal.isPending,
    /** Which sample line the voice says: a greeting where a character talks, else narration. */
    sampleLine: genreWording(genre) === "encounter" ? ("greeting" as const) : ("narration" as const),
    /** Why it could not be heard, as a translation key. */
    errorKey: failed,
    play,
    renew,
    stop,
  };
}

export type VoicePreview = ReturnType<typeof useVoicePreview>;
