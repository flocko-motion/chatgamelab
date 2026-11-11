import type { TFunction } from "i18next";

export const AI_QUALITY_TIERS = ["max", "high", "medium", "low"] as const;
export type AiQualityTier = (typeof AI_QUALITY_TIERS)[number];

/** Rank of a tier, higher = better; -1 for unknown tiers. */
function tierRank(tier: string): number {
  const index = AI_QUALITY_TIERS.indexOf(tier as AiQualityTier);
  return index < 0 ? -1 : AI_QUALITY_TIERS.length - index;
}

/** The tiers above limit, best first; none when limit is unset or unknown. */
export function tiersAboveLimit(limit?: string | null): AiQualityTier[] {
  if (!limit || tierRank(limit) < 0) return [];
  return AI_QUALITY_TIERS.filter((tier) => tierRank(tier) > tierRank(limit));
}

/**
 * Returns Select options for AI quality tiers.
 * @param t - i18next translation function (namespace: "common")
 * @param opts.includeEmpty - if true, prepends a "Server Default" / empty option
 * @param opts.availableTiers - if provided, tiers not in this set are disabled
 * @param opts.limit - the highest tier a free-use key is released up to: higher
 *   tiers are left out, and the empty option plays at the limit
 * @param opts.currentValue - a stored tier above the limit stays listed, disabled
 *   and marked as not released, so the select does not appear empty
 */
export function getAiQualityTierOptions(
  t: TFunction,
  opts?: {
    includeEmpty?: boolean;
    availableTiers?: string[];
    limit?: string | null;
    currentValue?: string | null;
  },
) {
  const available = opts?.availableTiers;
  const above = tiersAboveLimit(opts?.limit);
  const tiers = AI_QUALITY_TIERS.filter(
    (tier) => !above.includes(tier) || tier === opts?.currentValue,
  ).map((tier) =>
    above.includes(tier)
      ? {
          value: tier,
          label: t("aiQualityTier.limit.notReleasedOption", {
            tier: t(`aiQualityTier.${tier}`),
          }),
          disabled: true,
        }
      : {
          value: tier,
          label: t(`aiQualityTier.${tier}`),
          disabled: available ? !available.includes(tier) : false,
        },
  );

  if (opts?.includeEmpty) {
    // While a free-use key pays, no tier of one's own plays at its release.
    const emptyLabel =
      opts.limit && tierRank(opts.limit) >= 0
        ? t("aiQualityTier.limit.releaseDefault", {
            tier: t(`aiQualityTier.${opts.limit}`),
          })
        : t("aiQualityTier.serverDefault");
    return [{ value: "", label: emptyLabel }, ...tiers];
  }

  return tiers;
}
