import type { ParticipantListSort } from "@/api/hooks";
import { bothWays } from "@/api/lists";
import type { ListPreset } from "@/api/types";

/**
 * A workshop's participants, as its settings show them: embedded, without a
 * toolbar, so the query stays out of the URL. Its headers sort by name,
 * joining date and games.
 */
export const PARTICIPANT_LIST_PRESET: ListPreset<string, ParticipantListSort> = {
  sorts: [
    ...bothWays("createdAt", undefined, ["asc", "desc"]),
    ...bothWays("name", undefined, ["asc", "desc"]),
    ...bothWays("gamesCount"),
  ],
  defaultSort: { by: "createdAt", dir: "asc" },
  inUrl: false,
  defaultPageSize: 25,
};
