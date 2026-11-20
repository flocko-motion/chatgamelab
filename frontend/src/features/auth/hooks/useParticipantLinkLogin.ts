import { useEffect, useRef } from "react";
import { useNavigate } from "@tanstack/react-router";
import { participantLogin } from "@/providers/AuthProvider";
import { ROUTES } from "@/common/routes/routes";
import { buildShareUrl } from "@/common/lib/url";
import { toParticipantToken } from "@/common/lib/wordToken";

/**
 * Signs a participant in with the token from their link, once, then opens the
 * workshop; a token that doesn't sign in leads to the code page instead.
 */
export function useParticipantLinkLogin(token: string) {
  const navigate = useNavigate();
  const started = useRef(false);

  useEffect(() => {
    if (started.current) return;
    started.current = true;
    void participantLogin(toParticipantToken(token)).then((result) => {
      if (result === "ok") {
        // A full load, so the app starts with the new session.
        window.location.href = buildShareUrl(ROUTES.MY_WORKSHOP);
      } else {
        void navigate({ to: ROUTES.CODE as "/", replace: true });
      }
    });
  }, [token, navigate]);
}
