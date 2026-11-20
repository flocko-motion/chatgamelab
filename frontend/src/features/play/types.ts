export type GameFilter =
  | "all"
  | "own"
  | "public"
  | "organization"
  | "favorites"
  | "sponsored";

export type GameSortField =
  | "name"
  | "createdAt"
  | "modifiedAt"
  | "playCount"
  | "visibility"
  | "creator";
