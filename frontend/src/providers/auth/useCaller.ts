/**
 * Who is calling, from GET /api/users/me: anonymous, an account still
 * registering, or a registered account. Also completes registration.
 */
import { useCallback, useRef, useState } from "react";
import { api } from "@/api/client/api";
import type { User } from "@/api/generated/api.gen";
import { authLogger } from "@/config/logger";
import { ErrorCodes, extractRawErrorCode } from "@/common/types/errorCodes";
import i18n from "@/i18n";

export type CallerState = "unknown" | "anonymous" | "registering" | "registered";

export function useCaller() {
  const [state, setState] = useState<CallerState>("unknown");
  const [user, setUser] = useState<User | null>(null);
  const [error, setError] = useState<string | null>(null);
  const [isWorkshopInactive, setIsWorkshopInactive] = useState(false);

  // Ref for t, so fetchCaller keeps its identity when i18next re-renders.
  const tRef = useRef<(key: string) => string>((key) => key);
  const setTranslationRef = (t: (key: string) => string) => {
    tRef.current = t;
  };

  const applyUser = useCallback((next: User) => {
    setUser(next);
    setState("registered");
    if (next.language && next.language !== i18n.language) {
      authLogger.debug("Applying user language preference", {
        stored: next.language,
        current: i18n.language,
      });
      void i18n.changeLanguage(next.language);
    }
  }, []);

  /** Fetches who is calling; returns the state found. */
  const fetchCaller = useCallback(async (): Promise<CallerState> => {
    setError(null);
    try {
      const { data } = await api.getApiUsersMe();
      if (data.state === "registered" && data.user) {
        applyUser(data.user);
        return "registered";
      }
      setUser(null);
      const next = data.state === "registering" ? "registering" : "anonymous";
      setState(next);
      return next;
    } catch (err) {
      setUser(null);
      if (extractRawErrorCode(err) === ErrorCodes.AUTH_WORKSHOP_INACTIVE) {
        authLogger.info("Participant's workshop is inactive");
        setIsWorkshopInactive(true);
        setState("anonymous");
        return "anonymous";
      }
      authLogger.error("Failed to fetch the current user", { error: err });
      setError(tRef.current("errors.backendUserFetch"));
      setState("anonymous");
      return "anonymous";
    }
  }, [applyUser]);

  /** Completes registration of the signed-in account. */
  const register = useCallback(
    async (name: string, email: string, ageGroup: string) => {
      // The language the user registers in becomes their stored preference,
      // and game sessions are locked to it at launch.
      const { data } = await api.postApiAuthRegister({
        name,
        email: email || undefined,
        ageGroup,
        language: i18n.resolvedLanguage ?? i18n.language?.split("-")[0] ?? "en",
      });
      authLogger.info("User registered", { userId: data.id });
      applyUser(data);
    },
    [applyUser],
  );

  /** Forgets the caller, after signing out. */
  const clear = useCallback(() => {
    setUser(null);
    setState("anonymous");
    setError(null);
  }, []);

  return {
    state,
    user,
    error,
    isWorkshopInactive,
    fetchCaller,
    register,
    clear,
    setTranslationRef,
  };
}
