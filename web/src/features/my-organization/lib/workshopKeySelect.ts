/** DOM id of a workshop's API key select, so other controls can jump to it. */
export function workshopKeySelectId(workshopId: string) {
  return `workshop-api-key-${workshopId}`;
}

/** Scrolls to a workshop's API key select and focuses it. */
export function focusWorkshopKeySelect(workshopId: string) {
  const el = document.getElementById(workshopKeySelectId(workshopId));
  el?.scrollIntoView({ behavior: "smooth", block: "center" });
  el?.focus();
}
