import { useCallback, useEffect, useState } from "react";
import { useNavigate } from "@tanstack/react-router";
import {
  useCreateSession,
  useFetchShareInfo,
  useOpenSession,
  type NewSession,
} from "@/api/hooks";
import type { Mode } from "@/api/types";
import {
  problemFromError,
  unplayable,
  type Problem,
} from "@/common/lib/playability";
import { shareSessions } from "../lib/shareSessions";
import type { PlaybackTarget } from "../playback";

/** What a play route shows while the platform comes to a session, and once it has one. */
export type LaunchView =
  | { readonly kind: "working"; readonly text: "starting" | "opening" }
  /** The starter adapts the game's prompts before the session starts. */
  | { readonly kind: "advanced"; readonly game: string }
  /** The game plays live and in rounds, and the player chooses which. */
  | { readonly kind: "choose"; readonly what: NewSession }
  | {
      readonly kind: "refused";
      readonly problem: Problem | "offline";
      readonly when: "start" | "open";
    }
  | {
      readonly kind: "resumable";
      readonly id: string;
      /** Whether the link still has plays for a new session. */
      readonly startAnew: boolean;
      /** Whether starting anew counts against the link's plays. */
      readonly usesPlay: boolean;
    }
  | { readonly kind: "playing"; readonly id: string };

/** Refusals that say a remembered session is gone for good, so a share link forgets it. */
const GONE: ReadonlyArray<Problem> = ["unknown", "denied", "stale"];

/**
 * Comes to a session for a play route, on the platform: a game or a share
 * link gets a new session, which the address then names; a session is
 * opened before the playback-engine shows it, so a refusal is the
 * platform's to say. A share link's guest is offered to continue the
 * session this browser played through the link: continuing takes no play,
 * so it works without plays left; starting anew is offered only while there
 * are. Waits while auth is loading.
 */
export function usePlaybackLaunch(
  target: PlaybackTarget,
  language: string,
  ready: boolean,
) {
  const navigate = useNavigate();
  // A game started with advanced settings waits for its prompts.
  const advanced = target.kind === "game" && !!target.advanced;
  const { mutateAsync: create } = useCreateSession();
  const { mutateAsync: open } = useOpenSession();
  const fetchShareInfo = useFetchShareInfo();
  // Bumped to run the launch again, as Try again does.
  const [attempt, setAttempt] = useState(0);
  // What the launch came to, for the target and attempt it ran for; until
  // then the panel says it is under way.
  const key = `${target.kind}:${target.id}:${attempt}`;
  const [shown, setShown] = useState<{ key: string; view: LaunchView } | null>(
    null,
  );
  const view: LaunchView =
    shown?.key === key
      ? shown.view
      : {
          kind: "working",
          text: target.kind === "game" ? "starting" : "opening",
        };
  const setView = useCallback(
    (next: LaunchView) => setShown({ key, view: next }),
    [key],
  );

  /** Goes to a session the web app now has: the address names it, so a reload reopens it. */
  const toSession = useCallback(
    (id: string) => {
      justOpened.add(id);
      navigate({
        to: "/sessions/$sessionId",
        params: { sessionId: id },
        replace: true,
      });
    },
    [navigate],
  );

  /**
   * Creates a session and goes to it; for a share link it is remembered for
   * its token. Answers the refusal to show, or null once on the way.
   */
  const start = useCallback(
    async (what: NewSession): Promise<LaunchView | null> => {
      try {
        const id = await create({ what, language });
        if ("share" in what) shareSessions.remember(what.share, id);
        toSession(id);
        return null;
      } catch (thrown) {
        const problem = await problemFromError(thrown);
        if (problem === "choose_mode") return { kind: "choose", what };
        return { kind: "refused", problem, when: "start" };
      }
    },
    [create, language, toSession],
  );

  useEffect(() => {
    if (!ready) return;
    let cancelled = false;
    const show = (next: LaunchView | null) => {
      if (next && !cancelled) setView(next);
    };
    /** Opens a session: its id, or why not. */
    const opening = async (id: string): Promise<Problem | "offline" | null> => {
      try {
        await open(id);
        return null;
      } catch (thrown) {
        return problemFromError(thrown);
      }
    };

    async function launchSession(id: string): Promise<void> {
      if (justOpened.delete(id)) {
        show({ kind: "playing", id });
        return;
      }
      const refused = await opening(id);
      show(
        refused
          ? { kind: "refused", problem: refused, when: "open" }
          : { kind: "playing", id },
      );
    }

    async function launchShare(token: string): Promise<void> {
      let info;
      try {
        info = await fetchShareInfo(token);
      } catch (thrown) {
        // A token that opens nothing is not found: the link is not (any more) playable.
        show({
          kind: "refused",
          problem: thrown instanceof Response ? "denied" : "offline",
          when: "start",
        });
        return;
      }
      if (cancelled) return;
      const startRefused = unplayable(info);
      const remembered = shareSessions.get(token);
      if (remembered) {
        const refused = await opening(remembered);
        if (cancelled) return;
        if (!refused) {
          show({
            kind: "resumable",
            id: remembered,
            startAnew: !startRefused,
            usesPlay: info.remaining != null,
          });
          return;
        }
        if (!GONE.includes(refused as Problem)) {
          // Away for now (paused, unreachable, …): the session is still theirs.
          show({ kind: "refused", problem: refused, when: "open" });
          return;
        }
        shareSessions.forget(token);
      }
      if (startRefused) {
        show({ kind: "refused", problem: startRefused, when: "start" });
        return;
      }
      show({ kind: "working", text: "starting" });
      show(await start({ share: token }));
    }

    switch (target.kind) {
      case "game":
        // With advanced settings, the session starts once the prompts are set.
        if (advanced) show({ kind: "advanced", game: target.id });
        else void start({ game: target.id }).then(show);
        break;
      case "session":
        void launchSession(target.id);
        break;
      case "share":
        void launchShare(target.id);
        break;
    }
    return () => {
      cancelled = true;
    };
  }, [
    target.kind,
    target.id,
    advanced,
    ready,
    attempt,
    start,
    open,
    fetchShareInfo,
    setView,
  ]);

  return {
    view,
    continueSession: toSession,
    startAnew: () => {
      setView({ kind: "working", text: "starting" });
      void start({ share: target.id }).then(
        (refused) => refused && setView(refused),
      );
    },
    tryAgain: () => setAttempt((n) => n + 1),
    /** Starts the game the player chose to play live or in rounds. */
    choose: (mode: Mode) => {
      if (view.kind !== "choose") return;
      const what = { ...view.what, mode };
      setView({ kind: "working", text: "starting" });
      void start(what).then((refused) => refused && setView(refused));
    },
    /** Starts the game with its prompts adapted, from the advanced settings. */
    startWithPrompts: (prompts: Record<string, string>, mode?: Mode) => {
      setView({ kind: "working", text: "starting" });
      void start({ game: target.id, prompts, mode }).then(
        (refused) => refused && setView(refused),
      );
    },
  };
}

/**
 * Sessions this page just opened or created: going to one, the frame mounts
 * without opening it a second time. Each is used once.
 */
const justOpened = new Set<string>();
