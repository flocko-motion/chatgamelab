import { createContext, useContext } from "react";
import type { Branding } from "@/api/types";

/** The site's branding, as SiteProvider loaded it; null while unreachable. */
export const BrandingContext = createContext<Branding | null>(null);

/**
 * The site's branding: its name, operator, support address and links. An
 * empty link or address is one the admin hid. Empty everywhere while the
 * server cannot be reached.
 */
export function useBranding(): Branding {
  return useContext(BrandingContext) ?? EMPTY;
}

const EMPTY: Branding = {
  siteName: "",
  operatorName: "",
  operatorUrl: "",
  supportEmail: "",
  websiteUrl: "",
  contactUrl: "",
  privacyUrl: "",
  termsUrl: "",
  educatorInfoUrl: "",
  helpUrl: "",
  gameTipsUrl: "",
  apiKeyFaqUrl: "",
  aiInsightsUrl: "",
  reportGameUrl: "",
  creditName: "",
  creditUrl: "",
};
