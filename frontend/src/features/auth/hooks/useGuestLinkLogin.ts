import { useEffect, useRef } from "react";
import { useNavigate } from "@tanstack/react-router";
import { guestLogin } from "@/providers/AuthProvider";
import { ROUTES } from "@/common/routes/routes";
import { buildShareUrl } from "@/common/lib/url";
import { toGuestToken } from "@/common/lib/wordToken";

/**
 * Signs a Guest in a workshop in with the token from their link, once, then opens the
 * workshop; a token that doesn't sign in leads to the code page instead.
 */
export function useGuestLinkLogin(token: string) {
  const navigate = useNavigate();
  const started = useRef(false);

  useEffect(() => {
    if (started.current) return;
    started.current = true;
    void guestLogin(toGuestToken(token)).then((result) => {
      if (result === "ok") {
        // A full load, so the app starts with the new session.
        window.location.href = buildShareUrl(ROUTES.MY_WORKSHOP);
      } else {
        void navigate({ to: ROUTES.CODE as "/", replace: true });
      }
    });
  }, [token, navigate]);
}
