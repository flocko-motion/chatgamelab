import type { TranslationResources } from './types';
import en from './locales/en.json';
import de from './locales/de.json';

export const enResources: TranslationResources = en;
export const deResources: TranslationResources = de;

export const staticResources = {
  en: enResources,
  de: deResources,
};
