import { useEffect, useRef, useState } from "react";
import { Center, Loader } from "@mantine/core";
import { useNavigate } from "@tanstack/react-router";
import { useTranslation } from "react-i18next";
import { useAuth } from "@/providers/AuthProvider";
import { engineRoot, playbackURL, type PlaybackTarget } from "./playback";

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
  const signedInAs = useSignedInAs();
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
 * Who the frame plays as: undefined until auth has loaded, then the player's
 * id, or null for a guest. The session cookie reaches the playback-engine on
 * its own.
 */
function useSignedInAs(): string | null | undefined {
  const { isLoading, isAuthenticated, backendUser } = useAuth();
  if (isLoading) return undefined;
  return isAuthenticated ? (backendUser?.id ?? null) : null;
}
