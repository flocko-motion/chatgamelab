import { useCallback, useMemo } from "react";
import { useNavigate } from "@tanstack/react-router";

export type OrganisationTab = "members" | "workshops" | "apikeys" | "constraints";

export const ORGANISATION_TABS: readonly OrganisationTab[] = [
  "members",
  "workshops",
  "apikeys",
  "constraints",
];

interface UseOrganisationTabsOptions {
  /** The tab in the URL, if any. */
  tab?: OrganisationTab;
  /** The page was opened to create a workshop: the workshops tab opens. */
  autoCreateWorkshop?: boolean;
  /** Staff and heads see the workshops and keys. */
  canManage: boolean;
  /** Heads see the organisation's constraints. */
  canEditSettings: boolean;
}

/**
 * The organisation page's tab, kept in the URL. Each tab shows one list whose
 * query also sits in the URL (filter, search, order, page); switching tabs
 * starts the next list from its defaults, so one list's query never reaches
 * another.
 */
export function useOrganisationTabs({
  tab,
  autoCreateWorkshop,
  canManage,
  canEditSettings,
}: UseOrganisationTabsOptions) {
  const navigate = useNavigate();
  // The first is the default.
  const visible = useMemo(
    () =>
      ORGANISATION_TABS.filter(
        (t) =>
          t === "members" ||
          ((t === "workshops" || t === "apikeys") && canManage) ||
          (t === "constraints" && canEditSettings),
      ),
    [canManage, canEditSettings],
  );
  const wanted = tab ?? (autoCreateWorkshop ? "workshops" : undefined);
  const current = wanted && visible.includes(wanted) ? wanted : visible[0];

  const setTab = useCallback(
    (next: string | null) => {
      const value = ORGANISATION_TABS.find((t) => t === next);
      if (!value || value === current) return;
      navigate({
        to: ".",
        // Only the tab: the list keys (and a create action) are left behind.
        search: (() => ({ tab: value === visible[0] ? undefined : value })) as never,
      });
    },
    [current, navigate, visible],
  );

  return { tab: current, setTab, visible };
}
