import { useTranslation } from 'react-i18next';
import { useQuery } from '@tanstack/react-query';
import { isStaticLanguage } from '../../i18n/config';
import languagesData from '../../i18n/locales/languages.json';

export const useBackendTranslation = (namespace = 'common') => {
  const { t, i18n, ready } = useTranslation(namespace);
  
  // Load backend translations for non-static languages
  const { isLoading: isBackendLoading, error } = useQuery({
    queryKey: ['translations', i18n.language, namespace],
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
  
  const changeLanguage = async (language: string) => {
    await i18n.changeLanguage(language);
  };

  const availableLanguages = languagesData.map((lang) => ({
    code: lang.code,
    name: lang.label,
    isStatic: isStaticLanguage(lang.code),
  }));

  const currentLanguage = availableLanguages.find(lang => lang.code === i18n.language) || availableLanguages[0];

  return {
    currentLanguage,
    availableLanguages,
    changeLanguage,
  };
};
