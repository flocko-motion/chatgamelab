export interface CreateGameFormData {
  name: string;
  description: string;
  isPublic: boolean;
  systemMessageScenario?: string;
  systemMessageGameStart?: string;
  imageStyle?: string;
  /** How the game's voice sounds: the narrator's, or the character's. */
  voiceStyle?: string;
  statusFields?: string;
  /** One of the engine's genres; empty makes an adventure. */
  genre?: string;
  /** One of the playback-engine's themes; empty is the default look. */
  theme?: string;
}
