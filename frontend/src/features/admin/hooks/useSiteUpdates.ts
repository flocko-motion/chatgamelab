import { useEffect, useState } from "react";
import {
  checkHandled,
  checkMark,
  checkRefused,
  checkRunning,
  upgradeHandled,
  upgradeRunning,
  useCheckForUpdates,
  useRequestUpgrade,
  useUpdates,
  type CheckMark,
} from "@/api/hooks";

// The run host takes one check a minute.
const CHECK_INTERVAL_MS = 60_000;

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

  // The upgrade asked for here is installed: the server runs the new
  // release, this page still the old frontend. It loads the new one.
  const upgraded =
    awaiting !== null &&
    state?.install?.version === awaiting &&
    state.install.state === "installed" &&
    state.running === awaiting;
  useEffect(() => {
    if (!upgraded) return;
    const timer = setTimeout(() => window.location.reload(), 2000);
    return () => clearTimeout(timer);
  }, [upgraded]);
  const checking =
    checkRunning(state) || check.isPending || (checkAwaited !== null && !checkHandled(state, checkAwaited));

  // When a check is taken again: a minute after the last one finished, or
  // when the run host's Retry-After says.
  const [retryAt, setRetryAt] = useState<number | null>(null);
  const [refused, setRefused] = useState<"check_running" | "check_too_soon" | null>(null);
  const lastFinished = state?.check?.finished ? Date.parse(state.check.finished) : NaN;
  const nextCheckAt = Math.max(
    Number.isFinite(lastFinished) ? lastFinished + CHECK_INTERVAL_MS : 0,
    retryAt ?? 0,
  );

  // A ticking clock while a check is not taken yet, for the countdown.
  const [now, setNow] = useState(() => Date.now());
  const waitingForNextCheck = nextCheckAt > now;
  useEffect(() => {
    if (!waitingForNextCheck) return;
    const timer = setInterval(() => setNow(Date.now()), 1000);
    return () => clearInterval(timer);
  }, [waitingForNextCheck]);
  const checkInSeconds = waitingForNextCheck ? Math.ceil((nextCheckAt - now) / 1000) : 0;

  const checkNow = () => {
    const mark = checkMark(state);
    setRefused(null);
    check.mutate(undefined, {
      onSuccess: () => setCheckAwaited(mark),
      onError: (error) => {
        const r = checkRefused(error);
        if (!r) return;
        setRefused(r.code);
        setNow(Date.now());
        setRetryAt(Date.now() + (r.retryAfter ?? CHECK_INTERVAL_MS / 1000) * 1000);
      },
    });
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
    checkInSeconds,
    upgradedTo: upgraded ? awaiting : null,
    checkRefused: waitingForNextCheck ? refused : null,
    busy: installing || waiting || upgrade.isPending,
    requestUpgrade,
    checkNow,
  };
}
