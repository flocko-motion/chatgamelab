import i18n, { type Resource } from "i18next";
import { initReactI18next } from "react-i18next";
import LanguageDetector from "i18next-browser-languagedetector";

import { staticResources } from "./resources";
import { backendLoader } from "./backendLoader";
import { DEFAULT_LANGUAGE } from "./config";

i18n
  .use(backendLoader)
  .use(LanguageDetector)
  .use(initReactI18next)
  .init({
    fallbackLng: DEFAULT_LANGUAGE,
    load: "languageOnly", // Strip region code (en-US → en)
    debug: false,

    // Static resources for EN/DE
    resources: staticResources as unknown as Resource,

    // Backend loader for other languages
    backend: {
      loader: backendLoader,
    },

    // Language detection configuration
    detection: {
      order: ["localStorage", "navigator", "htmlTag"],
      caches: ["localStorage"],
    },

    interpolation: {
      escapeValue: false, // React already escapes
    },

    // React specific options
    react: {
      useSuspense: false, // Disable suspense mode for better control
    },
  });

export default i18n;
