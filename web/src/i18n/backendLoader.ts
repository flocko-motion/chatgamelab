import { isStaticLanguage } from './config';

// Backend loader for dynamic translations
export const backendLoader = {
  type: 'backend' as const,
  init: () => {},
  read: async (language: string, namespace: string): Promise<Record<string, string> | null> => {
    // Skip backend for static languages (en, de)
    if (isStaticLanguage(language)) {
      return null;
    }

    try {
      // Import the English translations for this namespace
      const { enResources } = await import('./resources');
      // eslint-disable-next-line @typescript-eslint/no-explicit-any
      const namespaceTranslations = (enResources as any)[namespace];
      
      if (!namespaceTranslations) {
        console.warn(`No translations found for namespace: ${namespace}`);
        return null;
      }

      // TODO: We have to replace this later with the actual translation endpoint
      // and actually do the translation
      console.log(`Translating ${namespace} from English to ${language}`);
      
      // const response = await apiClient.post('/translate', {
      //   sourceLanguage: 'en',
      //   targetLanguage: language,
      //   translations: namespaceTranslations,
      // });
      
      // For now, return the English translations as fallback
      return namespaceTranslations;
      
    } catch (error) {
      console.warn(`Failed to load translations for ${language}/${namespace}:`, error);
      return null; // Falls back to English
    }
  },
};

// Helper function to get English translations for a namespace
export async function getEnglishTranslations(namespace: string): Promise<Record<string, string>> {
  const { enResources } = await import('./resources');
  // eslint-disable-next-line @typescript-eslint/no-explicit-any
  return (enResources as any)[namespace] || {};
}
