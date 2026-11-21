package core

import (
	"go/parser"
	"go/token"
	"io/fs"
	"path/filepath"
	"sort"
	"strconv"
	"strings"
	"testing"
)

// rawAccess are the imports that reach the database directly: the generated
// queries, schemaf's raw connection and database/sql itself.
var rawAccess = map[string]bool{
	"cgl/db":                              true,
	"github.com/flocko-motion/schemaf/db": true,
	"database/sql":                        true,
}

// composition is the code that may import raw access for good: the
// composition root registers db.Provider.
var composition = map[string]bool{
	"app/app.go": true,
}

// legacy are the v1 files that still reach the database directly. The list
// only shrinks: porting an entity to the core removes its files here, and a
// file listed here that no longer imports raw access fails the test until
// its line goes too. Progress is deletion.
var legacy = map[string]bool{
	"api/auth.go":       true,
	"api/auth/login.go": true,
}

// Only the core reaches the database: no other code of the module imports
// raw access, except the composition root and the v1 files not yet ported.
// The engine is a module of its own and cannot import cgl/db.
func TestOnlyTheCoreReachesTheDatabase(t *testing.T) {
	root := ".." // the module, go/
	var sideDoors, ported []string
	reaching := map[string]bool{}

	err := filepath.WalkDir(root, func(path string, d fs.DirEntry, err error) error {
		if err != nil {
			return err
		}
		rel, _ := filepath.Rel(root, path)
		rel = filepath.ToSlash(rel)
		if d.IsDir() {
			switch rel {
			case "core", "db", "engine": // the core, the database itself, another module
				return filepath.SkipDir
			}
			return nil
		}
		if !strings.HasSuffix(rel, ".go") {
			return nil
		}
		f, err := parser.ParseFile(token.NewFileSet(), path, nil, parser.ImportsOnly)
		if err != nil {
			return err
		}
		for _, imp := range f.Imports {
			if p, _ := strconv.Unquote(imp.Path.Value); rawAccess[p] {
				reaching[rel] = true
			}
		}
		return nil
	})
	if err != nil {
		t.Fatal(err)
	}

	for file := range reaching {
		if !composition[file] && !legacy[file] {
			sideDoors = append(sideDoors, file)
		}
	}
	for file := range legacy {
		if !reaching[file] {
			ported = append(ported, file)
		}
	}
	sort.Strings(sideDoors)
	sort.Strings(ported)
	for _, f := range sideDoors {
		t.Errorf("%s reaches the database directly: go through the core", f)
	}
	for _, f := range ported {
		t.Errorf("%s no longer reaches the database: remove it from legacy", f)
	}
}
