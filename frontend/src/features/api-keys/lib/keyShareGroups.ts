import type { KeyShareView } from "@/api/types";

/** Where a share link draws its key from. */
export type ShareLinkSource = "organization" | "personal";

export interface ShareLinkEntry {
  share: KeyShareView;
  source: ShareLinkSource;
}

export interface KeyShareGroups {
  /** The key given to the owner's organisation. */
  organisations: KeyShareView[];
  /** Games the key sponsors publicly. */
  sponsorships: KeyShareView[];
  /** Share links paid for by the key. */
  shareLinks: ShareLinkEntry[];
  total: number;
}

/**
 * Groups the shares of one key by kind: given to an organisation, sponsoring
 * a game, or paying for a share link. The owner's own use (self) is left out.
 */
export function keyShareGroups(shares: KeyShareView[]): KeyShareGroups {
  const organisations = shares.filter(
    (s) => s.institution && !s.game && !s.isPrivateShare,
  );
  const sponsorships = shares.filter((s) => s.game && !s.isPrivateShare);
  const shareLinks = shares
    .filter((s) => s.isPrivateShare)
    .map((share): ShareLinkEntry => {
      const source = shares.find((s) => s.id === share.sourceShareId);
      return {
        share,
        source: source?.institution || share.institution
          ? "organization"
          : "personal",
      };
    });
  return {
    organisations,
    sponsorships,
    shareLinks,
    total: organisations.length + sponsorships.length + shareLinks.length,
  };
}
