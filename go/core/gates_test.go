package core

import (
	"go/ast"
	"go/parser"
	"go/token"
	"testing"

	"cgl/core/internal/rules"
)

// Every relation the grant table grants for an {action, entity} a gate covers
// must open a fragment of that gate's query. Otherwise the table promises a
// right the query silently ignores.
func TestGatesCoverTheGrantTable(t *testing.T) {
	for gate, gated := range gatesIn(t, "gates.go") {
		for pair := range pairsOf(gated) {
			for relation := range grantedRelations(pair.action, pair.entity) {
				if !gated[triplet{pair.action, pair.entity, relation}] {
					t.Errorf("%s: the table grants {%s, %s, %s}, but no gate opens it",
						gate, pair.action, pair.entity, relation)
				}
			}
		}
	}
}

type pair struct {
	action rules.Action
	entity rules.Entity
}

type triplet struct {
	action   rules.Action
	entity   rules.Entity
	relation rules.Relation
}

// gatesIn reads the granted(ctx, rules.A, rules.E, rules.R) calls of every
// function in file, as written.
func gatesIn(t *testing.T, file string) map[string]map[triplet]bool {
	t.Helper()
	f, err := parser.ParseFile(token.NewFileSet(), file, nil, 0)
	if err != nil {
		t.Fatalf("parse %s: %v", file, err)
	}
	gates := map[string]map[triplet]bool{}
	for _, decl := range f.Decls {
		fn, ok := decl.(*ast.FuncDecl)
		if !ok {
			continue
		}
		gated := map[triplet]bool{}
		ast.Inspect(fn, func(n ast.Node) bool {
			call, ok := n.(*ast.CallExpr)
			if !ok || len(call.Args) != 4 {
				return true
			}
			if id, ok := call.Fun.(*ast.Ident); !ok || id.Name != "granted" {
				return true
			}
			gated[triplet{
				rules.Action(constant(t, call.Args[1])),
				rules.Entity(constant(t, call.Args[2])),
				rules.Relation(constant(t, call.Args[3])),
			}] = true
			return true
		})
		if len(gated) > 0 {
			gates[fn.Name.Name] = gated
		}
	}
	return gates
}

// constant resolves rules.X, as written in gates.go, to the value of X.
func constant(t *testing.T, e ast.Expr) string {
	t.Helper()
	sel, ok := e.(*ast.SelectorExpr)
	if !ok {
		t.Fatalf("gate argument %T is not rules.X", e)
	}
	v, ok := vocabulary[sel.Sel.Name]
	if !ok {
		t.Fatalf("rules.%s is not in the test's vocabulary; add it", sel.Sel.Name)
	}
	return v
}

func pairsOf(gated map[triplet]bool) map[pair]bool {
	pairs := map[pair]bool{}
	for g := range gated {
		pairs[pair{g.action, g.entity}] = true
	}
	return pairs
}

// grantedRelations are the relations any role holds for {action, entity}.
func grantedRelations(action rules.Action, entity rules.Entity) map[rules.Relation]bool {
	relations := map[rules.Relation]bool{}
	for _, grants := range rules.Rules {
		for _, list := range grants {
			for _, g := range list {
				if g.Action == action && g.Entity == entity {
					relations[g.Relation] = true
				}
			}
		}
	}
	return relations
}

// vocabulary maps the names gates.go uses to their values.
var vocabulary = map[string]string{
	"List": string(rules.List), "Read": string(rules.Read),
	"Edit": string(rules.Edit), "Copy": string(rules.Copy), "Share": string(rules.Share),
	"Publish":  string(rules.Publish),
	"GameCard": string(rules.GameCard), "GameContent": string(rules.GameContent), "GameSettings": string(rules.GameSettings),
	"Public": string(rules.Public), "Own": string(rules.Own), "Everywhere": string(rules.Everywhere),
	"OfCurrentWorkshop": string(rules.OfCurrentWorkshop), "ViaShareLink": string(rules.ViaShareLink),
	"OwnOfCurrentWorkshop":    string(rules.OwnOfCurrentWorkshop),
	"PublicElsewhere":         string(rules.PublicElsewhere),
	"OfOwnOrganisation":       string(rules.OfOwnOrganisation),
	"Delete":                  string(rules.Delete),
	"UserCard":                string(rules.UserCard),
	"UserAccount":             string(rules.UserAccount),
	"UserContact":             string(rules.UserContact),
	"UserSecrets":             string(rules.UserSecrets),
	"MemberOfOwnOrganisation": string(rules.MemberOfOwnOrganisation),
	"GuestOfOwnOrganisation":  string(rules.GuestOfOwnOrganisation),
	"Create":                  string(rules.Create),
	"Suspend":                 string(rules.Suspend),
	"Organisation":            string(rules.Organisation),
}
