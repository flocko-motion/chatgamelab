---
id: test-vocabulary
applies_to:
  - "testing/**"
trigger: "adds or changes an integration test, or a word in testing/testutil"
---

# Integration tests speak a vocabulary

An integration test reads like plain English: who does what, and what the
platform answers. The words live in `testing/testutil`; the test files use
only those words.

- **A test file imports only** `testing`, testify, `github.com/google/uuid`,
  `cgl/testing/testutil` and harmless standard-library packages (`strings`,
  `fmt`, `regexp`, `time`, …). It never imports `net/http`, the generated
  client `cgl/apiclient`, the platform's packages (`cgl/api/...`, `cgl/obj`,
  `cgl/db`) or the engine's. `testing/vocabulary_test.go` enforces the list.
- **A missing word goes into `testutil`**, named so the test reads as a
  sentence (`alice.CreateWorkshop(…)`, `s.JoinWorkshop(token)`), in the file
  for its topic. A word calls the generated client `cgl/apiclient`; it never
  sends a hand-written request.
- **A type a test names is re-exported from `testutil`** (`types.go`, e.g.
  `type Game = apiclient.Game`), so a test checks what the API actually
  sends.
- **A refusal is asserted by its code**, from the API's error body
  `{"error": …, "code": …}`: `s.Forbidden(err)`, `s.NotFound(err)`,
  `s.Refused(err, testutil.CodeNoKey)`. "It failed" alone is no assertion.
- **A test asserts the right behaviour.** When the API answers something
  else, the test fails and the API gets fixed; the test isn't bent to the
  current answer.
