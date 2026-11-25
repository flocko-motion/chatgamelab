# translation

Translates the platform's UI texts from `en.json` and `de.json` into every
other language with OpenAI.

```sh
make translate          # go run ./translation — with OpenAI
make translate-claude   # go run ./translation --engine claude — with Claude Code
make translate-apply    # go run ./translation --apply — apply ticked proposals
make translate-accept   # go run ./translation --accept — record texts translated in the files
make translate-check    # go run ./translation --check — what CI runs
```

There are three ways to translate, all checked the same way:

- **OpenAI** (`make translate`) reads `OPENAI_API_KEY`; the model is fixed in
  `main.go`.
- **Claude Code** (`make translate-claude`) runs the local `claude` CLI on its
  user's login, no key. The tool checks each answer and hands what is wrong
  back to Claude, a few rounds per language. Claude also sees the language's
  current texts and may propose improvements where they are wrong (meaning,
  terms, grammar, English left over), each with its reason in English. Those
  are never applied unasked: they land in `translation/proposals.md` (not in
  git) as a checklist. Tick the ones to take and run `make translate-apply`;
  it applies them if the text did not change meanwhile, and drops the rest.
- **By hand, or by an agent** in the locale files directly, then
  `make translate-accept`. It checks every field that needs translating and
  records them as translated only when all are right.

The checks: every field is there, with the English `{{placeholders}}`, and no
language holds a field en.json no longer has.

- `frontend/src/i18n/locales` holds `en.json` and `de.json`, written by
  hand, and every translation.
- `go/lang/locales` is the server's embedded copy of that
  directory.
- `languages.json` is written from the language list in `main.go`. The web app
  and the server both read their languages from it.

Before translating, `en.json` and `de.json` must have the same fields. A field
in one only is added to the other as `TODO: TRANSLATE`, to fill in by hand.

A field en.json no longer has is dropped from every language, without
calling OpenAI; `--check` fails while one is left.

Each language has a `<lang>.hash.json` beside its `<lang>.json`, with a hash of
the English and German text of every field. A run sends only the fields whose
hash changed, in both source languages. Delete a language's hash file to
translate it from scratch.

To add a language, add it to the list in `main.go` and run `make translate`.
