/** Whether the workshop-mode info was dismissed for good, in this browser. */
const STORAGE_KEY = "cgl_workshop_mode_info_dismissed";

/** Whether to show the workshop-mode info when entering workshop mode. */
export function shouldShowInfoModal(): boolean {
  try {
    return localStorage.getItem(STORAGE_KEY) !== "true";
  } catch {
    return true;
  }
}

/** Never shows the workshop-mode info again in this browser. */
export function dismissInfoModal(): void {
  try {
    localStorage.setItem(STORAGE_KEY, "true");
  } catch {
    // Not kept: the info shows again next time.
  }
}

/** Set on entering workshop mode, for this tab, until the info has had its chance to show. */
const PENDING_KEY = "cgl_workshop_mode_pending";

/** Marks workshop mode as just entered, so the info shows once the workshop is loaded, across a reload. */
export function markWorkshopModeEntered(): void {
  try {
    sessionStorage.setItem(PENDING_KEY, "true");
  } catch {
    // Not kept: the info does not show this time.
  }
}

/** Whether workshop mode was just entered; the mark is used up. */
export function takeWorkshopModeEntered(): boolean {
  try {
    const pending = sessionStorage.getItem(PENDING_KEY) !== null;
    sessionStorage.removeItem(PENDING_KEY);
    return pending;
  } catch {
    return false;
  }
}
