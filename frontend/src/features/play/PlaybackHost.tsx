import { useEffect, useRef, useState } from "react";
import { Center, Loader } from "@mantine/core";
import { useNavigate } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { useAuth } from "@/providers/AuthProvider";
import { useAuthenticatedApi } from "@/api/useAuthenticatedApi";
import { engineRoot, playbackURL, type PlaybackTarget } from "./playback";

/** How long before the play cookie expires it is renewed. */
const RENEW_AHEAD_MS = 5 * 60 * 1000;
/** How soon a failed renewal is tried again. */
const RETRY_MS = 60 * 1000;

interface Frame {
  readonly target: PlaybackTarget;
  readonly language: string;
  /** The session the frame reported creating, which the page's URL now names. */
  readonly reported: string | null;
}

function sameTarget(a: PlaybackTarget, b: PlaybackTarget): boolean {
  return a.kind === b.kind && a.id === b.id;
}

/**
 * The frame every play route renders: the playback-engine in an iframe, and
 * nothing of the platform's own. It stays mounted while the address changes
 * from a play route to the session the playback-engine created, so the frame
 * is not reloaded.
 */
export function PlaybackHost({ target }: { target: PlaybackTarget }) {
  const { i18n } = useTranslation();
  const navigate = useNavigate();
  const signedInAs = usePlayCookie();
  const frameRef = useRef<HTMLIFrameElement>(null);

  const [frame, setFrame] = useState<Frame>({
    target,
    language: i18n.language,
    reported: null,
  });
  // A route the frame did not report, such as Back to another session, opens anew.
  const followsFrame =
    sameTarget(frame.target, target) ||
    (target.kind === "session" && target.id === frame.reported);
  if (!followsFrame) {
    setFrame({ target, language: i18n.language, reported: null });
  }

  useEffect(() => {
    const origin = engineRoot().origin;
    const onMessage = (event: MessageEvent) => {
      if (event.origin !== origin) return;
      if (event.source !== frameRef.current?.contentWindow) return;
      const data = event.data as { type?: unknown; id?: unknown } | null;
      if (data?.type !== "session" || typeof data.id !== "string") return;
      const id = data.id;
      setFrame((current) => ({ ...current, reported: id }));
      navigate({
        to: "/sessions/$sessionId",
        params: { sessionId: id },
        replace: true,
      });
    };
    window.addEventListener("message", onMessage);
    return () => window.removeEventListener("message", onMessage);
  }, [navigate]);

  if (signedInAs === undefined) {
    return (
      <Center style={{ flex: 1 }}>
        <Loader size="lg" />
      </Center>
    );
  }

  const src = playbackURL(frame.target, frame.language);
  return (
    <iframe
      key={src}
      ref={frameRef}
      src={src}
      title="ChatGameLab"
      allow="microphone; autoplay; clipboard-write; fullscreen"
      style={{ flex: 1, width: "100%", border: 0, display: "block" }}
    />
  );
}

/**
 * Gives the playback-engine the signed-in player's identity: sets the play
 * cookie before the frame mounts and renews it while the page is open. Answers
 * undefined until the frame may mount, then the player's id, or null for a
 * guest.
 */
function usePlayCookie(): string | null | undefined {
  const { isLoading, isAuthenticated, backendUser } = useAuth();
  const api = useAuthenticatedApi();
  const userId = isAuthenticated ? (backendUser?.id ?? null) : null;
  const [issuedFor, setIssuedFor] = useState<string | null>(null);

  useEffect(() => {
    if (isLoading || !userId || !api) return;
    let cancelled = false;
    let timer: ReturnType<typeof setTimeout> | undefined;
    const issue = async () => {
      let next = RETRY_MS;
      try {
        const { data } = await api.auth.playTokenCreate();
        const expiresAt = data.expiresAt ? Date.parse(data.expiresAt) : NaN;
        if (!Number.isNaN(expiresAt)) {
          next = Math.max(expiresAt - Date.now() - RENEW_AHEAD_MS, RETRY_MS);
        }
      } catch {
        // The frame mounts regardless and acts as a guest; the playback-engine
        // explains whatever it is refused.
      }
      if (cancelled) return;
      setIssuedFor(userId);
      timer = setTimeout(issue, next);
    };
    void issue();
    return () => {
      cancelled = true;
      clearTimeout(timer);
    };
  }, [isLoading, userId, api]);

  if (isLoading) return undefined;
  if (!userId) return null;
  return issuedFor === userId ? userId : undefined;
}
