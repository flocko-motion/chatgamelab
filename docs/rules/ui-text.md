---
id: ui-text
applies_to:
  - "frontend/src/**/*.tsx"
  - "frontend/src/**/*.ts"
excludes:
  - "frontend/src/i18n/**"
  - "frontend/src/api/generated/**"
trigger: "shows text to the user, or adds or changes a translation key"
---

# Text the user sees is translated

Every text a user can see goes through `t()`; the app runs in many languages,
and a literal in a component shows up untranslated in all but one.

```tsx
// Bad
<Button>Save</Button>
<HelperText>{"Hello " + name}</HelperText>

// Good
const { t } = useTranslation('common');
<Button>{t('buttons.save')}</Button>
<HelperText>{t('greeting', { name })}</HelperText>
```

- **Keys say what the text is for** and sit in the namespace they belong to
  (`common`, `auth`, `dashboard`, `errors`, `myWorkshop`, `navigation`):
  `profile.saveButton`, not `button3`.
- **Dynamic values are interpolated**, never concatenated: word order differs
  between languages.
- **`en.json` and `de.json` are written by hand**; every other language is
  generated from them (`make translate`) and never edited by hand. That every
  key is present and translated is checked by CI (`make translate-check`).

Not user-facing, so not translated: log messages, test ids, CSS classes, keys
and values of API requests.
