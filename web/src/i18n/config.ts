export const DEFAULT_LANGUAGE = 'en';

export const STATIC_LANGUAGES = ['en', 'de'] as const;

export type StaticLanguage = typeof STATIC_LANGUAGES[number];

export const isStaticLanguage = (language: string): language is StaticLanguage => {
  return (STATIC_LANGUAGES as readonly string[]).includes(language);
};

export const NAMESPACES = ['common', 'navigation', 'game', 'errors', 'auth', 'dashboard'] as const;

export type Namespace = typeof NAMESPACES[number];
