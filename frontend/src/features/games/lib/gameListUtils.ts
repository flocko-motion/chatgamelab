/**
 * Get a date label for a game based on the current sort field.
 * Shows createdAt when sorting by createdAt, otherwise modifiedAt.
 */
export function getGameDateLabel(
  game: { meta?: { createdAt?: string; modifiedAt?: string } },
  sortField: string,
): string | undefined {
  const dateValue = gameListDate(game, sortField);
  return dateValue ? new Date(dateValue).toLocaleDateString() : undefined;
}

/**
 * The date a game list shows for a game: when it was created if the list is
 * sorted by that, else when it was last changed.
 */
export function gameListDate(
  game: { meta?: { createdAt?: string; modifiedAt?: string } },
  sortField: string,
): string | undefined {
  return sortField === "createdAt" ? game.meta?.createdAt : game.meta?.modifiedAt;
}

/**
 * Hands the browser a file to save.
 */
export function downloadBlob(blob: Blob, fileName: string) {
  const url = URL.createObjectURL(blob);
  const link = document.createElement("a");
  link.href = url;
  link.download = fileName;
  document.body.appendChild(link);
  link.click();
  document.body.removeChild(link);
  URL.revokeObjectURL(url);
}

/**
 * Saves a game's .cgl file, its YAML and its cover, under the game's name.
 */
export function downloadGameFile(file: Blob, gameName: string | undefined) {
  downloadBlob(file, `${gameName || "game"}.cgl`);
}
