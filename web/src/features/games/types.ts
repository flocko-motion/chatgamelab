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
}
