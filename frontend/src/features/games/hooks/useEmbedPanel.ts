import { useState } from "react";
import { useTranslation } from "react-i18next";
import { useGameEmbed, useSetGameEmbed, useVerifiedPages, useWorkshop } from "@/api/hooks";
import type { GameView } from "@/api/types";
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
 * toggle, the organisation's verified page it runs on, the key that pays,
 * and the snippet to copy. Embedding is for organisations, on their
 * verified page (the spec's Embedding), so the panel is offered for a game
 * of an organisation only and always sets the verified mode. Each change is
 * saved as soon as it is complete; until then (no key, no page) it stays
 * here. The server decides who may embed (mayShare) and which pages are the
 * organisation's; the panel only shows what it answers.
 */
export function useEmbedPanel(game: GameView) {
  const { t } = useTranslation("common");
  const shown = game.mayShare && !!game.workshopId;
  const { data: embed, isLoading } = useGameEmbed(game.id, shown);
  const save = useSetGameEmbed(game.id);
  const { keys, loading: keysLoading } = usePayingKeys();

  // Verified pages are the game's organisation's: its workshop's. The list
  // is answered to the organisation's head and staff only.
  const { data: workshop } = useWorkshop(game.workshopId);
  const organisation = workshop?.institution?.id;
  const { data: pages } = useVerifiedPages(organisation);

  const [draft, setDraft] = useState<Choices | null>(null);
  const saved: Choices = {
    on: embed?.enabled ?? false,
    shareId: embed?.sponsorKeyShareId ?? null,
    pageId: embed?.verifiedPageId ?? null,
  };
  const choices = draft ?? saved;

  const change = (patch: Partial<Choices>) => {
    const next = { ...choices, ...patch };
    // An organisation has one verified page: choosing the mode chooses it.
    if (!next.pageId && pages?.length === 1) next.pageId = pages[0].id;

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

  // A page set earlier stays shown, even where the caller may not list them.
  const pageOptions = (pages ?? []).map((p) => ({ value: p.id, label: p.pageUrl }));
  if (embed?.verifiedPageId && !pageOptions.some((p) => p.value === embed.verifiedPageId)) {
    pageOptions.unshift({ value: embed.verifiedPageId, label: embed.verifiedPageUrl ?? "" });
  }

  const live = !!embed?.enabled && !draft;
  const snippet = live
    ? embedSnippet({
        loaderUrl: new URL("playback-engine/embed.js", engineRoot()).href,
        token: embed.token,
        verified: true,
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
    pageId: choices.pageId,
    /** The organisation's pages the game may run on, as the server lists them. */
    pageOptions,
    keyGroups: groups,
    noKeys: !keysLoading && keyOptions.length === 0,
    /** On, but not saved yet: a key or a page is still to be chosen. */
    incomplete: choices.on && !live && !save.isPending,
    /** The snippet for the host page, while the saved link is on. */
    snippet,
    /** Saved and on, but its page still waits for an admin's approval: nothing starts yet. */
    waitingForApproval: live && !embed?.verifiedPageApproved,
    setOn: (on: boolean) => change({ on }),
    setShare: (shareId: string | null) => change({ shareId }),
    setPage: (pageId: string | null) => change({ pageId }),
  };
}

export type EmbedPanel = ReturnType<typeof useEmbedPanel>;
