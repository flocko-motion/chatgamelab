# Incremental Hash-Based Translation System

## Overview

The translation system now uses a hash-based incremental approach to only translate fields that have actually changed in the source (English + German) content. This dramatically reduces API token usage on subsequent translation runs.

## How It Works

### 1. Hash Generation
For each target language (e.g., `fr.json`), we generate a corresponding hash file (`fr.hash.json`) that contains:
- A hash for each field path in the JSON structure
- The hash is computed from the concatenated English + German values for that field

### 2. Change Detection
When running translation:
1. Compute current hashes from `en.json` + `de.json`
2. Load existing hash file for target language (if exists)
3. Compare hashes to identify changed fields
4. Extract only the changed fields into a reduced JSON structure

### 3. Incremental Translation
- Only the reduced JSON (changed fields) is sent to the AI for translation
- This can reduce token usage by 90%+ on subsequent runs where only a few fields changed

### 4. Merge Back
- The partial translation is merged back into the existing full translation
- The updated hash file is saved for next run

## File Structure

```
web/src/i18n/locales/
├── en.json           # Source (English)
├── de.json           # Source (German)
├── fr.json           # Target translation (French)
├── fr.hash.json      # Hash file for French (tracks what was translated)
├── es.json           # Target translation (Spanish)
├── es.hash.json      # Hash file for Spanish
└── ...
```

## Usage

```bash
# Translate all languages (only changed fields)
./cgl lang translate --platform openai --input ../web/src/i18n/locales --output ../web/src/i18n/locales

# Translate specific language
./cgl lang translate --platform openai --lang fr --input ../web/src/i18n/locales --output ../web/src/i18n/locales

# Parallel translation with 3 threads
./cgl lang translate --platform openai --threads 3 --input ../web/src/i18n/locales --output ../web/src/i18n/locales
```

## Output Example

```
✓ Original files have matching structure
✓ No TODO placeholders found
Computed hashes for 1246 fields
Translating to 35 language(s) using openai platform (unlimited parallelism)...

⏭ French (fr): up-to-date, skipping
📝 Spanish (es): 12/1246 fields changed
📝 Italian (it): 12/1246 fields changed
⏭ German (de): up-to-date, skipping

⏳ Translating 2 language(s)...

✓ Spanish (es): translated 12 fields → ./lang/locales/es.json
✓ Italian (it): translated 12 fields → ./lang/locales/it.json

✓ Translation complete: 37/37 successful (tokens: 450 in, 520 out, 970 total)
```

## Architecture

### Generic Utilities (functional package)
- `CollectFieldValues()` - Extract all leaf values from JSON
- `ExtractFieldsByPaths()` - Extract specific fields by path
- `MergeJSON()` - Merge two JSON structures
- `ComputeHash()` - Compute SHA256 hash

### Translation-Specific (lang package)
- `ComputeFieldHashes()` - Compute hashes for all fields from multiple sources
- `ExtractChangedFields()` - Extract only fields that changed
- `MergeTranslations()` - Merge partial translation into full translation
- `SaveHashFile()` - Save hash map to JSON file

## Benefits

1. **Token Savings**: Only translate what changed (90%+ reduction on incremental runs)
2. **Faster**: Less data to translate means faster completion
3. **Cost Effective**: Dramatically lower API costs for maintaining translations
4. **Incremental**: Can update translations as source evolves without re-translating everything
5. **Safe**: Hash files track exactly what was translated, preventing drift

## Migration

Existing translation files work as-is. On first run with the new system:
- Hash files will be created for all languages
- All fields will be translated (since no hash file exists yet)
- Subsequent runs will be incremental

To force re-translation of a language, simply delete its `.hash.json` file.
