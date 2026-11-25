import { useState } from "react";
import { useSite, useUpdateSiteBranding } from "@/api/hooks";
import type { Branding } from "@/api/types";

export type BrandingField = keyof Branding;

/** The branding fields the admin edits, in groups as the form shows them. */
export const BRANDING_GROUPS: { id: string; fields: BrandingField[] }[] = [
  { id: "identity", fields: ["siteName", "operatorName", "operatorLegalName", "operatorUrl", "supportEmail"] },
  {
    id: "links",
    fields: [
      "websiteUrl",
      "contactUrl",
      "privacyUrl",
      "termsUrl",
      "educatorInfoUrl",
      "helpUrl",
      "gameTipsUrl",
      "apiKeyFaqUrl",
      "aiInsightsUrl",
      "reportGameUrl",
    ],
  },
  { id: "credit", fields: ["creditName", "creditUrl"] },
];

/**
 * The site's branding as the admin edits it: each field's value and default,
 * saving one field at a time. A field saved as its default goes back to
 * following the default.
 */
export function useSiteBrandingForm() {
  const { data, isLoading } = useSite();
  const update = useUpdateSiteBranding();

  const value = (field: BrandingField) => data?.branding[field] ?? "";
  const defaultOf = (field: BrandingField) => data?.defaults[field] ?? "";
  const isDefault = (field: BrandingField) => value(field) === defaultOf(field);

  const [errors, setErrors] = useState<Partial<Record<BrandingField, BrandingError>>>({});
  const setError = (field: BrandingField, error: BrandingError | undefined) =>
    setErrors((all) => ({ ...all, [field]: error }));

  const save = (field: BrandingField, next: string) => {
    const trimmed = next.trim();
    const error = check(field, trimmed);
    setError(field, error);
    if (error || trimmed === value(field)) return;
    update.mutate({ [field]: trimmed === defaultOf(field) ? null : trimmed });
  };
  const reset = (field: BrandingField) => {
    setError(field, undefined);
    update.mutate({ [field]: null });
  };

  return {
    loading: isLoading,
    saving: update.isPending,
    value,
    defaultOf,
    isDefault,
    error: (field: BrandingField) => errors[field],
    save,
    reset,
  };
}

/** Why a value does not fit its field: the server refuses the same. */
export type BrandingError = "nameRequired" | "notEmail" | "notUrl";

function check(field: BrandingField, value: string): BrandingError | undefined {
  if (field === "siteName") return value ? undefined : "nameRequired";
  if (!value) return undefined;
  if (field === "supportEmail") return /^[^\s@]+@[^\s@]+\.[^\s@]+$/.test(value) ? undefined : "notEmail";
  if (field.endsWith("Url")) {
    try {
      const url = new URL(value);
      return (url.protocol === "http:" || url.protocol === "https:") && url.host ? undefined : "notUrl";
    } catch {
      return "notUrl";
    }
  }
  return undefined;
}
