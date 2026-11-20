export type SortField =
  | "name"
  | "createdAt"
  | "modifiedAt"
  | "playCount"
  | "visibility"
  | "creator";

export interface CreateGameFormData {
  name: string;
  description: string;
  isPublic: boolean;
  systemMessageScenario?: string;
  systemMessageGameStart?: string;
  imageStyle?: string;
  statusFields?: string;
  /** One of the engine's genres; empty makes an adventure. */
  genre?: string;
  /** One of the playback-engine's themes; empty is the default look. */
  theme?: string;
}
