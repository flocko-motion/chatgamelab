# translation

Translates the platform's UI texts from `en.json` and `de.json` into every
other language with OpenAI.

```sh
make translate         # go run ./translation
make translate-check   # go run ./translation --check — what CI runs
```

It reads `OPENAI_API_KEY`. Everything else is fixed in `main.go`: the model,
the language list, and the paths.

- `platform/web/src/i18n/locales` holds `en.json` and `de.json`, written by
  hand, and every translation.
- `platform/server/lang/locales` is the server's embedded copy of that
  directory.
- `languages.json` is written from the language list in `main.go`. The web app
  and the server both read their languages from it.

Before translating, `en.json` and `de.json` must have the same fields. A field
in one only is added to the other as `TODO: TRANSLATE`, to fill in by hand.

Each language has a `<lang>.hash.json` beside its `<lang>.json`, with a hash of
the English and German text of every field. A run sends only the fields whose
hash changed, in both source languages. Delete a language's hash file to
translate it from scratch.

To add a language, add it to the list in `main.go` and run `make translate`.
