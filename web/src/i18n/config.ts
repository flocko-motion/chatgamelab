export const DEFAULT_LANGUAGE = 'en';

export const STATIC_LANGUAGES = [
  'ar', 'bar', 'bg', 'bs', 'cs', 'da', 'de', 'el', 'en', 'es',
  'fa', 'fi', 'fr', 'hi', 'hr', 'hu', 'id', 'it', 'ja', 'ko',
  'nl', 'no', 'pl', 'ps', 'pt', 'ro', 'ru', 'sk', 'sl', 'so',
  'sq', 'sr', 'sv', 'ti', 'tr', 'uk', 'zh'
] as const;

export type StaticLanguage = typeof STATIC_LANGUAGES[number];

export const isStaticLanguage = (language: string): language is StaticLanguage => {
  return (STATIC_LANGUAGES as readonly string[]).includes(language);
};

export const NAMESPACES = ['common', 'navigation', 'game', 'errors', 'auth', 'dashboard'] as const;

export type Namespace = typeof NAMESPACES[number];
