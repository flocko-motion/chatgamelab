import type { TFunction } from "i18next";

export const AI_QUALITY_TIERS = ["max", "high", "medium", "low"] as const;
export type AiQualityTier = (typeof AI_QUALITY_TIERS)[number];

/**
 * Returns Select options for AI quality tiers.
 * @param t - i18next translation function (namespace: "common")
 * @param opts.includeEmpty - if true, prepends a "Server Default" / empty option
 * @param opts.availableTiers - if provided, tiers not in this set are disabled
 */
export function getAiQualityTierOptions(
  t: TFunction,
  opts?: { includeEmpty?: boolean; availableTiers?: string[] },
) {
  const available = opts?.availableTiers;
  const tiers = [
    { value: "max", label: t("aiQualityTier.max"), disabled: available ? !available.includes("max") : false },
    { value: "high", label: t("aiQualityTier.high"), disabled: available ? !available.includes("high") : false },
    { value: "medium", label: t("aiQualityTier.medium"), disabled: available ? !available.includes("medium") : false },
    { value: "low", label: t("aiQualityTier.low"), disabled: available ? !available.includes("low") : false },
  ];

  if (opts?.includeEmpty) {
    return [{ value: "", label: t("aiQualityTier.serverDefault") }, ...tiers];
  }

  return tiers;
}
