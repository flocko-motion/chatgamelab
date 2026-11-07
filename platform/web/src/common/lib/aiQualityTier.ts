import type { TFunction } from "i18next";
import type { EnginePlatformInfo } from "@/api/generated";

/** The engine's tiers, cheapest first. A tier decides quality, never which features exist. */
export const AI_QUALITY_TIERS = ["low", "medium", "high"] as const;
export type AiQualityTier = (typeof AI_QUALITY_TIERS)[number];

/** A stored tier as one on offer: "max" reads as "high"; empty stays empty. */
export function normalizeAiQualityTier(
  tier: string | null | undefined,
): string {
  if (!tier) return "";
  return tier === "max" ? "high" : tier;
}

/** The tier's label: economy, balanced or high. */
export function getAiQualityTierLabel(
  t: (key: string) => string,
  tier: string,
): string {
  return t(`aiQualityTier.${normalizeAiQualityTier(tier)}`);
}

/**
 * Returns Select options for AI quality tiers.
 * @param t - i18next translation function (namespace: "common")
 * @param opts.includeEmpty - if true, prepends a "Server Default" / empty option
 * @param opts.platform - if given, offers that platform's tiers from the engine's catalogue
 */
export function getAiQualityTierOptions(
  t: TFunction,
  opts?: { includeEmpty?: boolean; platform?: EnginePlatformInfo },
) {
  const ids: string[] = opts?.platform?.tiers
    ?.map((tier) => tier.id ?? "")
    .filter(Boolean) ?? [...AI_QUALITY_TIERS];
  const tiers = ids.map((id) => ({
    value: id,
    label: getAiQualityTierLabel(t, id),
  }));

  if (opts?.includeEmpty) {
    return [{ value: "", label: t("aiQualityTier.serverDefault") }, ...tiers];
  }

  return tiers;
}
