import languagesData from "./locales/languages.json";

export interface LanguageOption {
  code: string;
  name: string;
}

export const LANGUAGE_OPTIONS: LanguageOption[] = languagesData.map((lang) => ({
  code: lang.code,
  name: lang.label,
}));

const languageNameByCode = new Map(
  LANGUAGE_OPTIONS.map((language) => [language.code.toLowerCase(), language.name]),
);

export function getNativeLanguageName(languageCode?: string | null): string {
  if (!languageCode) return "";

  const normalizedCode = languageCode.toLowerCase();
  const baseCode = normalizedCode.split("-")[0];

  return (
    languageNameByCode.get(normalizedCode) ||
    languageNameByCode.get(baseCode) ||
    languageCode
  );
}
