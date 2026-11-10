import type { TFunction } from "i18next";

/**
 * What the workshop key selection shows while the workshop has no key of its
 * own: the key it falls back to (organisation, then server), or that none does.
 * @param source - ObjWorkshop.apiKeySource
 */
export function workshopApiKeyFallbackLabel(
  t: TFunction,
  source: string | undefined,
): string {
  switch (source) {
    case "organization":
      return t("myOrganization.workshops.apiKeyFallbackOrganization");
    case "server":
      return t("myOrganization.workshops.apiKeyFallbackServer");
    default:
      return t("myOrganization.workshops.apiKeyFallbackNone");
  }
}

/** Relabels the "no key" option of a key Select with what applies instead. */
export function labelEmptyKeyOption<T>(options: T[], label: string): T[] {
  return options.map((option) =>
    option && typeof option === "object" && "value" in option && option.value === ""
      ? { ...option, label }
      : option,
  );
}
