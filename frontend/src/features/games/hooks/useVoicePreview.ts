import { useCallback, useEffect, useRef, useState } from "react";
import { useNewVoice, useVoiceSample, type VoiceSample } from "@/api/hooks";
import { extractRawErrorCode } from "@/common/types/errorCodes";
import { genreWording } from "../lib/editorPages";

/** A sample as played: for which genre and voice style, and its audio. */
interface Heard {
  key: string;
  audio: Blob;
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
 * Hearing a game's own voice in the editor: the sample the voice made for
 * the voice style says, played here (the overview's tile and the Sound page
 * share one), and making a new voice for the style. A sample heard once
 * plays again without asking the server, until the style changes. Only a
 * voice set to custom is heard (custom): with a built-in choice the session
 * casts its own voice, which a sample would misrepresent. Without a voice
 * style the built-in voice speaks.
 */
export function useVoicePreview(genre: string, voiceStyle: string, scenario: string, custom: boolean) {
  const style = voiceStyle.trim();
  const key = `${genre}\n${style}`;

  const sample = useVoiceSample();
  const renewal = useNewVoice();
  const [heard, setHeard] = useState<Heard | null>(null);
  const [playing, setPlaying] = useState(false);
  const [failed, setFailed] = useState<string | null>(null);
  // Played through an audio context unlocked by the click itself: making a
  // voice takes longer than a browser keeps a click's permission to play,
  // so a player started once the sample arrives would stay silent.
  const context = useRef<AudioContext | null>(null);
  const source = useRef<AudioBufferSourceNode | null>(null);

  const stop = useCallback(() => {
    const node = source.current;
    source.current = null;
    if (node) {
      node.onended = null;
      node.stop();
    }
    setPlaying(false);
  }, []);

  /** Unlocks playing, while the click that asks for it is still handled. */
  const unlock = () => {
    context.current ??= new AudioContext();
    void context.current.resume();
    return context.current;
  };

  const start = async (ac: AudioContext, audio: Blob) => {
    stop();
    try {
      const buffer = await ac.decodeAudioData(await audio.arrayBuffer());
      const node = ac.createBufferSource();
      node.buffer = buffer;
      node.connect(ac.destination);
      node.onended = () => {
        if (source.current === node) stop();
      };
      source.current = node;
      setPlaying(true);
      node.start();
    } catch {
      setPlaying(false);
      setFailed("games.editor.sound.errors.failed");
    }
  };

  const keep = (got: VoiceSample) => {
    const next = { key, audio: got.audio };
    setHeard(next);
    return next.audio;
  };

  /** Plays the sample, or stops it while it plays. */
  const play = async () => {
    if (playing) return stop();
    const ac = unlock();
    setFailed(null);
    if (heard?.key === key) return start(ac, heard.audio);
    try {
      await start(ac, keep(await sample.mutateAsync({ genre, voice: "custom", voiceStyle: style, scenario })));
    } catch (error) {
      setFailed(errorKey(error));
    }
  };

  /** Replaces the voice for the style on the paying key, and plays the new one. */
  const renew = async () => {
    stop();
    const ac = unlock();
    setFailed(null);
    try {
      await start(ac, keep(await renewal.mutateAsync({ genre, voice: "custom", voiceStyle: style, scenario })));
    } catch (error) {
      setFailed(errorKey(error));
    }
  };

  // The sample stops, and its audio goes, when the editor closes.
  useEffect(
    () => () => {
      source.current?.stop();
      void context.current?.close();
    },
    [],
  );

  return {
    /** A voice is set to custom: a voice of its own per key and style, heard and remade here. */
    custom,
    /** The game has a voice style; without one the built-in voice speaks. */
    hasStyle: style !== "",
    /** A new voice can be made: the genre makes them, and there is a style. */
    mayRenew: custom && style !== "",
    playing,
    /** The sample is being fetched or made. */
    loading: sample.isPending,
    /** A new voice is being made. */
    renewing: renewal.isPending,
    /** Which sample line the voice says: a greeting where a character talks, else narration. */
    sampleLine: genreWording(genre) === "adventure" ? ("narration" as const) : ("greeting" as const),
    /** Why it could not be heard, as a translation key. */
    errorKey: failed,
    play,
    renew,
    stop,
  };
}

export type VoicePreview = ReturnType<typeof useVoicePreview>;
