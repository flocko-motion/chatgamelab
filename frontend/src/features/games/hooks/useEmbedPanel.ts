import { useEffect, useRef, useState } from "react";
import { useTranslation } from "react-i18next";
import { useGameEmbed, useGenres, useSetGameEmbed, useVerifiedPages } from "@/api/hooks";
import type { GameView, Mode } from "@/api/types";
import { engineRoot } from "@/features/play/playback";
import { embedSnippet } from "../lib/embedSnippet";
import { usePayingKeys } from "./usePayingKeys";

/** The panel's choices: on or off, what pays, which page. */
interface Choices {
  on: boolean;
  shareId: string | null;
  pageId: string | null;
}

/**
 * Embedding a game on another website, as the Sharing page sets it: the
 * toggle, the key that pays, and the snippet to copy. It runs on the
 * organisation's verified page, its only one, so there is no page to choose;
 * the snippet shows once the link is saved with that page and a key. Embedding is for organisations, on their
 * verified page (the spec's Embedding), so the panel always sets the
 * verified mode. It is shown wherever the game may be shared, and says what
 * is to be done first while embedding cannot work yet (notices). Each change
 * is saved as soon as it is complete; until then (no key, no page) it stays
 * here. The server decides who may embed (mayShare) and which pages are the
 * organisation's; the panel only shows what it answers.
 */
export function useEmbedPanel(game: GameView) {
  const { t } = useTranslation("common");
  const organisation = game.organisationId;
  const shown = game.mayShare;
  const { data: embed, isLoading } = useGameEmbed(game.id, shown);
  const save = useSetGameEmbed(game.id);
  const { keys, loading: keysLoading } = usePayingKeys();
  // Live or rounds, for a game whose genre plays both: a part of the snippet
  // only, so the host decides, and none leaves the choice to the player.
  const { data: genres } = useGenres();
  const modes = genres?.find((g) => g.id === game.genre)?.subgenres.map((s) => s.mode) ?? [];
  const [mode, setMode] = useState<Mode | null>(null);
  const preselected = mode && modes.includes(mode) ? mode : undefined;

  // Verified pages are the game's organisation's, as the server names it
  // (its workshop's, or its creator's). The list is answered to the
  // organisation's head and staff only.
  const { data: pages, isError: pagesHidden } = useVerifiedPages(organisation);

  const [draft, setDraft] = useState<Choices | null>(null);
  const saved: Choices = {
    on: embed?.enabled ?? false,
    shareId: embed?.sponsorKeyShareId ?? null,
    pageId: embed?.verifiedPageId ?? null,
  };
  const choices = draft ?? saved;

  // The organisation's one verified page, where the caller may list it.
  const orgPage = pages?.[0];
  // The saved page still exists: the organisation's, or, where the caller
  // may not list pages, the one the server names.
  const savedPageExists = !!embed?.verifiedPageId && (pagesHidden || embed.verifiedPageId === orgPage?.id);

  const change = (patch: Partial<Choices>) => {
    const next = { ...choices, ...patch };
    // An organisation has one verified page: the link always takes it, also
    // in place of one removed since.
    if (orgPage) next.pageId = orgPage.id;

    if (!next.on) {
      setDraft(null);
      // Off keeps the link and its token, as it was set.
      if (embed?.enabled) {
        save.mutate({ enabled: false, mode: embed.mode, verifiedPageId: embed.verifiedPageId });
      }
      return;
    }
    setDraft(next);
    const complete = !!next.shareId && !!next.pageId;
    if (!complete) return;
    save.mutate(
      {
        enabled: true,
        mode: "verified",
        sponsorKeyShareId: next.shareId ?? undefined,
        verifiedPageId: next.pageId ?? undefined,
      },
      { onSettled: () => setDraft(null) },
    );
  };

  // The key paying now may be one the caller cannot choose (someone else's).
  const keyOptions = keys.map((k) => ({
    value: k.shareId,
    label: `${k.name} (${k.platform})`,
    group: k.isOrg ? t("games.privateShare.sourceOrg") : t("games.privateShare.sourcePersonal"),
  }));
  if (embed?.sponsorKeyShareId && !keys.some((k) => k.shareId === embed.sponsorKeyShareId)) {
    keyOptions.unshift({
      value: embed.sponsorKeyShareId,
      label: `${embed.keyName} (${embed.platform})`,
      group: t("games.editor.embed.payingNow"),
    });
  }
  const groups = [...new Set(keyOptions.map((o) => o.group))].map((group) => ({
    group,
    items: keyOptions.filter((o) => o.group === group).map(({ value, label }) => ({ value, label })),
  }));

  // A link left on a page removed since moves to the organisation's page
  // once, keeping its key: there is nothing to choose.
  const moved = useRef(false);
  useEffect(() => {
    if (moved.current || !embed?.enabled || !embed.sponsorKeyShareId || !orgPage) return;
    if (embed.verifiedPageId === orgPage.id) return;
    moved.current = true;
    save.mutate({
      enabled: true,
      mode: "verified",
      sponsorKeyShareId: embed.sponsorKeyShareId,
      verifiedPageId: orgPage.id,
    });
  }, [embed, orgPage, save]);

  // Live: saved on, with an existing page and a key. Only then the snippet.
  const live = !!embed?.enabled && savedPageExists && !!embed.sponsorKeyShareId && !draft;

  // The page it runs on: the organisation's, or the one the server names.
  const page = orgPage;
  const pageUrl = orgPage?.pageUrl ?? (savedPageExists ? embed?.verifiedPageUrl : undefined);

  // What is to be done first, in order. The first two leave nothing to
  // embed on; a missing key leaves nothing to pay with. A page waiting for
  // approval can be set up already.
  const noOrganisation = !organisation;
  const noPages = !noOrganisation && !!pages && !pagesHidden && pages.length === 0;
  const noKeys = !keysLoading && keyOptions.length === 0;
  const waitingForApproval =
    !noOrganisation && !noPages && (live ? !embed?.verifiedPageApproved : !!page && !page.approvedAt);
  const blocked = noOrganisation || noPages || noKeys;
  const snippet = live && embed
    ? embedSnippet({
        loaderUrl: new URL("playback-engine/embed.js", engineRoot()).href,
        token: embed.token,
        verified: true,
        mode: preselected,
        placeholders: {
          css: t("games.editor.embed.snippetCss"),
          player: t("games.editor.embed.snippetPlayer"),
          request: t("games.editor.embed.snippetRequest"),
        },
      })
    : null;

  return {
    /** The caller may embed the game, by the server's mayShare. */
    shown,
    loading: isLoading || keysLoading,
    saving: save.isPending,
    on: choices.on,
    shareId: choices.shareId,
    /** The address of the verified page the game runs on, once there is one. */
    pageUrl,
    keyGroups: groups,
    noKeys,
    /** The game belongs to no organisation: embedding is for organisations' games. */
    noOrganisation,
    /** The organisation has no verified page yet: one is to be registered first. */
    noPages,
    /** Nothing to embed on or to pay with yet: the toggle cannot turn it on. */
    blocked,
    /** On, but not live yet: the key is still to be chosen. */
    incomplete: choices.on && !live && !save.isPending,
    /** The snippet for the host page: only while the saved link is on, with its page and key. */
    snippet,
    /** Its page still waits for an admin's approval: nothing starts there yet. */
    waitingForApproval,
    /** The halves a host may preselect, live first: a choice only where there are two. */
    modes: modes.length > 1 ? modes : [],
    /** The half the snippet preselects; null leaves the choice to the player. */
    mode: preselected ?? null,
    setMode,
    setOn: (on: boolean) => change({ on }),
    setShare: (shareId: string | null) => change({ shareId }),
  };
}

export type EmbedPanel = ReturnType<typeof useEmbedPanel>;
