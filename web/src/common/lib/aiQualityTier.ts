import type { TFunction } from "i18next";

export const AI_QUALITY_TIERS = ["high", "medium", "low"] as const;
export type AiQualityTier = (typeof AI_QUALITY_TIERS)[number];

/**
 * Returns Select options for AI quality tiers.
 * @param t - i18next translation function (namespace: "common")
 * @param opts.includeEmpty - if true, prepends a "Server Default" / empty option
 */
export function getAiQualityTierOptions(
  t: TFunction,
  opts?: { includeEmpty?: boolean },
) {
  const tiers = [
    { value: "high", label: t("aiQualityTier.high") },
    { value: "medium", label: t("aiQualityTier.medium") },
    { value: "low", label: t("aiQualityTier.low") },
  ];

  if (opts?.includeEmpty) {
    return [{ value: "", label: t("aiQualityTier.serverDefault") }, ...tiers];
  }

  return tiers;
}
