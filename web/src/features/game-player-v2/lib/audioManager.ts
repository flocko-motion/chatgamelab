/**
 * Global audio manager — ensures only one audio source plays at a time.
 *
 * When any audio action starts (playback or recording), all other active
 * sources are stopped first. Components register a stop callback via
 * `registerAudioSource` and call `stopAllAudio` before starting their own.
 *
 * Usage (playback):
 *   stopAllAudio();
 *   const unregister = registerAudioSource(() => { audio.pause(); });
 *   audio.play();
 *   // later: unregister();
 *
 * Usage (recording):
 *   stopAllAudio();
 *   const unregister = registerAudioSource(() => { recorder.stop(); });
 *   recorder.start();
 *   // later: unregister();
 */

type StopFn = () => void;

const activeSources = new Set<StopFn>();

/** Register an active audio source. Returns an unregister function. */
export function registerAudioSource(stop: StopFn): () => void {
  activeSources.add(stop);
  return () => {
    activeSources.delete(stop);
  };
}

/** Stop all currently active audio sources (playback and recording). */
export function stopAllAudio(): void {
  activeSources.forEach((stop) => stop());
  activeSources.clear();
}
