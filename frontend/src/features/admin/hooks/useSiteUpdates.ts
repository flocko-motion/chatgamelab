import { useState } from "react";
import {
  checkHandled,
  checkMark,
  checkRunning,
  upgradeHandled,
  upgradeRunning,
  useCheckForUpdates,
  useRequestUpgrade,
  useUpdates,
  type CheckMark,
} from "@/api/hooks";

/**
 * The site's updates for the admin page: what runs, what it may upgrade to,
 * and the last upgrade and check. While an upgrade or a check is asked for or
 * running, the state is polled; during an upgrade the server restarts, and
 * the poll picks it up again.
 */
export function useSiteUpdates() {
  // The last version asked for.
  const [awaiting, setAwaiting] = useState<string | null>(null);
  // The state when the last check was asked for.
  const [checkAwaited, setCheckAwaited] = useState<CheckMark | null>(null);

  const query = useUpdates(awaiting, checkAwaited);
  const state = query.data;
  const upgrade = useRequestUpgrade();
  const check = useCheckForUpdates();

  const installing = upgradeRunning(state);
  const waiting = awaiting !== null && !upgradeHandled(state, awaiting);
  const checking =
    checkRunning(state) || check.isPending || (checkAwaited !== null && !checkHandled(state, checkAwaited));

  const checkNow = () => {
    const mark = checkMark(state);
    check.mutate(undefined, { onSuccess: () => setCheckAwaited(mark) });
  };

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
    lastCheck: state?.check,
    checking,
    busy: installing || waiting || upgrade.isPending,
    requestUpgrade,
    checkNow,
  };
}
