import { useTranslation } from "react-i18next";
import { useQuery } from "@tanstack/react-query";
import { queryKeys } from "@/api/queryKeys";
import { useAuthenticatedApi } from "@/api/useAuthenticatedApi";
import { apiLogger } from "@/config/logger";
import { isStaticLanguage, DEFAULT_LANGUAGE } from "../../i18n/config";
import { LANGUAGE_OPTIONS } from "../../i18n/languages";

export const useBackendTranslation = (namespace = "common") => {
  const { t, i18n, ready } = useTranslation(namespace);

  // Load backend translations for non-static languages
  const { isLoading: isBackendLoading, error } = useQuery({
    queryKey: queryKeys.translations(i18n.language, namespace),
    queryFn: async () => {
      // Only load from backend for non-static languages
      if (isStaticLanguage(i18n.language)) {
        return null;
      }

      // Force reload resources for the current language and namespace
      await i18n.reloadResources(i18n.language, namespace);
      return true;
    },
    enabled: !isStaticLanguage(i18n.language) && !ready,
    retry: 2,
  });

  const changeLanguage = async (language: string) => {
    await i18n.changeLanguage(language);
  };

  return {
    t,
    i18n,
    ready,
    isLoading: isBackendLoading || !ready,
    isBackendLanguage: !isStaticLanguage(i18n.language),
    changeLanguage,
    error,
  };
};

// Hook for language switching with loading states
export const useLanguageSwitcher = () => {
  const { i18n } = useTranslation();
  const api = useAuthenticatedApi();

  const changeLanguage = async (language: string) => {
    await i18n.changeLanguage(language);
    // Persist language preference to backend (best-effort, don't block UI)
    if (api) {
      api.users.meLanguagePartialUpdate({ language }).catch((err) => {
        apiLogger.warning("Failed to persist language preference", {
          error: err,
        });
      });
    }
  };

  const allLanguages = LANGUAGE_OPTIONS.map((lang) => ({
    code: lang.code,
    name: lang.name,
    isStatic: isStaticLanguage(lang.code),
  }));

  // Sort: English, Deutsch first, then separator, then others alphabetically
  const priorityLanguages = allLanguages.filter(
    (lang) => lang.code === "en" || lang.code === "de",
  );
  const otherLanguages = allLanguages
    .filter((lang) => lang.code !== "en" && lang.code !== "de")
    .sort((a, b) => a.name.localeCompare(b.name));

  const availableLanguages = [
    ...priorityLanguages.sort((a) => (a.code === "en" ? -1 : 1)), // en first, then de
    { code: "__separator__", name: "───────────", isStatic: false },
    ...otherLanguages,
  ];

  // Extract base language code (e.g., 'en' from 'en-US')
  const baseLanguageCode = i18n.language.split("-")[0];

  // Find current language by exact match or base code match, fallback to DEFAULT_LANGUAGE
  const currentLanguage =
    allLanguages.find((lang) => lang.code === i18n.language) ||
    allLanguages.find((lang) => lang.code === baseLanguageCode) ||
    allLanguages.find((lang) => lang.code === DEFAULT_LANGUAGE) ||
    allLanguages[0];

  return {
    currentLanguage,
    availableLanguages,
    changeLanguage,
  };
};
