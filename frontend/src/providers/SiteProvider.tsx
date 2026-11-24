import { useEffect, type ReactNode } from "react";
import i18n from "i18next";
import { useSite } from "@/api/hooks";
import type { Branding } from "@/api/types";
import { BrandingContext } from "./branding";

/**
 * Loads how the site presents itself before the app renders: texts name the
 * site and its operator through `{{siteName}}` and `{{operatorName}}`, and the
 * browser tab shows the site's name.
 */
export function SiteProvider({ children }: { children: ReactNode }) {
  const { data, isError } = useSite();
  const branding = data?.branding ?? null;

  if (branding) applyBranding(branding);

  useEffect(() => {
    if (branding) document.title = branding.siteName;
  }, [branding]);

  // Without the branding, texts would show their placeholders; the app waits
  // for it, and renders without it only if the server cannot be reached.
  if (!branding && !isError) return null;
  return <BrandingContext.Provider value={branding}>{children}</BrandingContext.Provider>;
}

/** Sets the names texts fill in; read by every `t()` from now on. */
function applyBranding(b: Branding) {
  const interpolation = i18n.options.interpolation ?? {};
  interpolation.defaultVariables = {
    ...interpolation.defaultVariables,
    siteName: b.siteName,
    operatorName: b.operatorName,
  };
  i18n.options.interpolation = interpolation;
}
