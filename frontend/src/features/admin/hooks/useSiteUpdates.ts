import { useState } from "react";
import { upgradeHandled, upgradeRunning, useRequestUpgrade, useUpdates } from "@/api/hooks";

/**
 * The site's updates for the admin page: what runs, what it may upgrade to,
 * and the last upgrade. While an upgrade is asked for or running, the state
 * is polled; the server restarts in between, and the poll picks it up again.
 */
export function useSiteUpdates() {
  // The last version asked for.
  const [awaiting, setAwaiting] = useState<string | null>(null);

  const query = useUpdates(awaiting);
  const state = query.data;
  const upgrade = useRequestUpgrade();

  const installing = upgradeRunning(state);
  const waiting = awaiting !== null && !upgradeHandled(state, awaiting);

  const requestUpgrade = (version: string) =>
    upgrade.mutate(version, { onSuccess: () => setAwaiting(version) });

  return {
    loading: query.isLoading,
    running: state?.running ?? "",
    checked: state?.checked,
    updaterEnabled: state !== undefined && state.status !== "updater not enabled",
    available: state?.available ?? [],
    install: state?.install,
    request: state?.request,
    busy: installing || waiting || upgrade.isPending,
    requestUpgrade,
  };
}
