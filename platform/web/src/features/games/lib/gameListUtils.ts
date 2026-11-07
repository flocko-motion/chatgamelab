import type { ObjGame } from "@/api/generated";

/**
 * Get a date label for a game based on the current sort field.
 * Shows createdAt when sorting by createdAt, otherwise modifiedAt.
 */
export function getGameDateLabel(
  game: ObjGame,
  sortField: string,
): string | undefined {
  const dateValue =
    sortField === "createdAt" ? game.meta?.createdAt : game.meta?.modifiedAt;
  return dateValue ? new Date(dateValue).toLocaleDateString() : undefined;
}

/**
 * Trigger a YAML file download in the browser.
 */
export function downloadYamlFile(yaml: string, gameName: string | undefined) {
  const blob = new Blob([yaml], { type: "application/x-yaml" });
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = `${gameName || "game"}.yaml`;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}
