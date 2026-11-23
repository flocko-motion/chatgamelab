import { api } from "../client/api";
import { useQuery, useMutation, useQueryClient } from "@tanstack/react-query";
import { handleApiError } from "@/config/queryClient";
import { queryKeys } from "../queryKeys";
import type {
  GenreInfo,
  PlatformInfo,
  ThemeInfo,
  SystemSettings,
  Error as ApiError,
  GetVersionResp,
  UpdateSystemSettingsReq,
  SystemConstraintsResp,
  SchemafUpdateState,
} from "../generated/api.gen";

// Platforms hook (requires authentication)
export function usePlatforms() {

  return useQuery<PlatformInfo[], ApiError>({
    queryKey: queryKeys.platforms,
    queryFn: () =>
      api.getApiPlatforms().then((response) => response.data),
  });
}


// The engine's genres, each with the features it needs, for the game editor
export function useGenres() {

  return useQuery<GenreInfo[], ApiError>({
    queryKey: queryKeys.genres,
    queryFn: () => api.getApiGenres().then((response) => response.data),
    staleTime: Infinity,
  });
}

// The playback-engine's looks, the default first, for the game editor
export function useThemes() {

  return useQuery<ThemeInfo[], ApiError>({
    queryKey: queryKeys.themes,
    queryFn: () => api.getApiThemes().then((response) => response.data),
    staleTime: Infinity,
  });
}

// System Settings hook (requires authentication)
export function useSystemSettings() {

  return useQuery<SystemSettings, ApiError>({
    queryKey: queryKeys.systemSettings,
    queryFn: () => api.getApiSystemSettings().then((response) => response.data),
  });
}

// Version hook (public endpoint, no auth needed)
export function useVersion() {

  return useQuery<GetVersionResp, ApiError>({
    queryKey: queryKeys.version,
    queryFn: () => api.getApiVersion().then((response) => response.data),
  });
}

// Update System Settings mutation (admin only)
export function useUpdateSystemSettings() {
  const queryClient = useQueryClient();

  return useMutation<
    SystemSettings,
    ApiError,
    UpdateSystemSettingsReq
  >({
    mutationFn: (request) =>
      api.patchApiSystemSettings(request)
        .then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.systemSettings });
    },
  });
}

// Set/clear system free-use API key (admin only)
export function useSetSystemFreeUseKey() {
  const queryClient = useQueryClient();

  return useMutation<
    SystemSettings,
    ApiError,
    { apiKeyId: string | null }
  >({
    mutationFn: ({ apiKeyId }) =>
      api.patchApiSystemSettingsFreeUseKey({
          apiKeyId: apiKeyId ?? undefined,
        })
        .then((response) => response.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.systemSettings });
    },
    onError: handleApiError,
  });
}

/** The platform's own prompt constraints; loaded only while `enabled`. */
export function useSystemConstraints(enabled = true) {
  return useQuery<SystemConstraintsResp, ApiError>({
    queryKey: queryKeys.systemConstraints,
    queryFn: () => api.getApiSystemConstraints().then((r) => r.data),
    enabled,
  });
}

const UPDATES_POLL_MS = 5000;

/** Whether an upgrade runs on the site. */
export function upgradeRunning(state: SchemafUpdateState | undefined): boolean {
  return state?.status === "installing" || state?.install?.state === "installing";
}

/**
 * Whether the run host has handled a request for `version`: installed it (or
 * rolled back), or rejected the request.
 */
export function upgradeHandled(state: SchemafUpdateState | undefined, version: string): boolean {
  if (!state || upgradeRunning(state)) return false;
  return (
    state.install?.version === version ||
    (state.request?.version === version && state.request.state === "rejected")
  );
}

/** Whether the run host is looking for new releases. */
export function checkRunning(state: SchemafUpdateState | undefined): boolean {
  return state?.status === "checking";
}

/** What a check asked for is compared against: the state when it was asked. */
export interface CheckMark {
  checked?: string;
  finished?: string;
}

/** The mark of `state`, taken when a check is asked for. */
export function checkMark(state: SchemafUpdateState | undefined): CheckMark {
  return { checked: state?.checked, finished: state?.check?.finished };
}

/**
 * Whether the run host has handled a check asked for at `mark`: it finished a
 * check since (found releases or none, failed, or refused it).
 */
export function checkHandled(state: SchemafUpdateState | undefined, mark: CheckMark): boolean {
  if (!state || checkRunning(state)) return false;
  return state.checked !== mark.checked || state.check?.finished !== mark.finished;
}

/**
 * The site's running version, the versions it may upgrade to and the state of
 * the last upgrade and check. Polled while an upgrade or a check runs, and
 * while the upgrade to `awaiting` or the check asked for at `checkAwaited` is
 * not handled yet: the server restarts during an upgrade, so failed polls are
 * expected then.
 */
export function useUpdates(awaiting: string | null = null, checkAwaited: CheckMark | null = null) {
  return useQuery<SchemafUpdateState, ApiError>({
    queryKey: queryKeys.updates,
    queryFn: () => api.getApiUpdates().then((r) => r.data),
    refetchInterval: (query) => {
      const data = query.state.data;
      const waiting =
        (awaiting !== null && !upgradeHandled(data, awaiting)) ||
        (checkAwaited !== null && !checkHandled(data, checkAwaited));
      return waiting || upgradeRunning(data) || checkRunning(data) ? UPDATES_POLL_MS : false;
    },
    retry: awaiting !== null ? true : 3,
  });
}

/** The run host refused a check for now: one runs, or the last is too recent. */
export interface CheckRefused {
  code: "check_running" | "check_too_soon";
  /** Seconds until a check is taken again, from the answer's Retry-After. */
  retryAfter?: number;
}

/** A refused check, out of the error a check request failed with; null for any other error. */
export function checkRefused(error: unknown): CheckRefused | null {
  if (!error || typeof error !== "object") return null;
  const r = error as { error?: { code?: string }; headers?: Headers };
  const code = r.error?.code;
  if (code !== "check_running" && code !== "check_too_soon") return null;
  const seconds = Number(r.headers?.get?.("Retry-After"));
  return { code, retryAfter: Number.isFinite(seconds) && seconds > 0 ? seconds : undefined };
}

/**
 * Asks the run host to look for new releases now; it fetches and verifies,
 * never installs. A refusal for now (checkRefused) is the caller's to show;
 * any other error goes to the error modal.
 */
export function useCheckForUpdates() {
  const queryClient = useQueryClient();
  return useMutation<void, ApiError, void>({
    mutationFn: () => api.postApiUpdatesCheck().then(() => undefined),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.updates });
    },
    onError: (error) => {
      if (!checkRefused(error)) handleApiError(error);
    },
  });
}

/** Asks the run host to install `version`; it checks the version again. */
export function useRequestUpgrade() {
  const queryClient = useQueryClient();
  return useMutation<{ version: string }, ApiError, string>({
    mutationFn: (version) =>
      api.postApiUpdatesRequest({ version }).then((r) => r.data),
    onSuccess: () => {
      queryClient.invalidateQueries({ queryKey: queryKeys.updates });
    },
    onError: handleApiError,
  });
}
