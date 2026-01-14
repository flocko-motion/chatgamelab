/* eslint-disable @typescript-eslint/no-explicit-any */
// Translation resources type - loosely typed to avoid constant sync issues with JSON files
// The actual structure is defined in the JSON files (en.json, de.json)
export type TranslationResources = Record<string, any>;

export type TranslationKey = string;
