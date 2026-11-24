import { useExportGameFile } from "@/api/hooks";
import type { GameView } from "@/api/types";
import { downloadGameFile } from "../lib";

/** Downloads a game as a .cgl file; the mutation shows what goes wrong. */
export function useDownloadGame() {
  const exportFile = useExportGameFile();
  return {
    downloading: exportFile.isPending,
    run: async (game: Pick<GameView, "id" | "name">) => {
      try {
        downloadGameFile(await exportFile.mutateAsync(game.id), game.name);
      } catch {
        // shown by the mutation's error handler
      }
    },
  };
}
