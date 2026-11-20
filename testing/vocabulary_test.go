package testing

import (
	"go/parser"
	"go/token"
	"path/filepath"
	"strconv"
	"testing"
)

// allowedImports is what a test file may import: the testing tools, ids, the
// vocabulary in testutil, and harmless standard-library packages. Everything
// that reaches the API goes through testutil (docs/rules/test-vocabulary.md).
var allowedImports = map[string]bool{
	"testing":                             true,
	"github.com/stretchr/testify/suite":   true,
	"github.com/stretchr/testify/assert":  true,
	"github.com/stretchr/testify/require": true,
	"github.com/google/uuid":              true,
	"cgl/testing/testutil":                true,
	"fmt":                                 true,
	"strings":                             true,
	"regexp":                              true,
	"time":                                true,
	"sort":                                true,
	"slices":                              true,
	"strconv":                             true,
}

// Every test file speaks only the vocabulary: no HTTP, no generated client,
// no platform or engine packages.
func TestTestsUseOnlyTheVocabulary(t *testing.T) {
	files, err := filepath.Glob("*_test.go")
	if err != nil {
		t.Fatal(err)
	}
	if len(files) == 0 {
		t.Fatal("no test files found")
	}
	fset := token.NewFileSet()
	for _, file := range files {
		if file == "vocabulary_test.go" { // the checker itself parses Go
			continue
		}
		f, err := parser.ParseFile(fset, file, nil, parser.ImportsOnly)
		if err != nil {
			t.Fatalf("parsing %s: %v", file, err)
		}
		for _, imp := range f.Imports {
			path, err := strconv.Unquote(imp.Path.Value)
			if err != nil {
				t.Fatalf("%s: import %s: %v", file, imp.Path.Value, err)
			}
			if !allowedImports[path] {
				t.Errorf("%s imports %q: tests speak only the vocabulary in testutil; add a word there (docs/rules/test-vocabulary.md)", fset.Position(imp.Pos()), path)
			}
		}
	}
}
