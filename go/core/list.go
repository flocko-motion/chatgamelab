package core

import (
	"cmp"
	"context"
	"slices"
	"strings"
)

// Every list works the same way: the caller asks with a ListQuery, the list
// answers with a Page. The query holds wishes, not rights: it narrows what
// the gates let through, never widens it. A list names what it knows
// (ListSpec); a wish it does not know is refused, naming the known ones.

// ListQuery is what the caller asks of a list.
type ListQuery struct {
	Filter  []string // combined, each narrows; none for all
	Search  string   // part of the list's searched texts, any case
	SortBy  string   // one of the list's orders; empty for its default
	SortDir string   // asc, desc; empty for the default order's direction
	Limit   int      // page size; 0 for all
	Offset  int      // matches to skip
}

// Page is one page of a list, and how many items match across all pages.
type Page[T any] struct {
	Items []T
	Total int
}

// ListSpec is what a list knows: its filters, its orders, and the order it
// takes when none is asked for.
type ListSpec struct {
	Filters []string
	// FilterFamilies are prefixes that each name a family of filters, e.g.
	// "platform:" for one filter per platform.
	FilterFamilies []string
	Sorts          []string
	DefaultSort    string
	DefaultDir     string // asc, desc
}

// Lister is a list of T: what it knows, and its one query for a page.
type Lister[T any] interface {
	Spec() ListSpec
	// Query answers a query already checked against the spec, with its
	// defaults applied.
	Query(ctx context.Context, q ListQuery) (Page[T], error)
}

// List is the one way into a list: it refuses a wish the list does not
// know, naming the known ones, applies the list's default order, and runs
// the list's query.
func List[T any](ctx context.Context, l Lister[T], q ListQuery) (Page[T], error) {
	spec := l.Spec()
	if err := spec.check(q); err != nil {
		return Page[T]{}, err
	}
	if q.SortBy == "" {
		q.SortBy = spec.DefaultSort
		if q.SortDir == "" {
			q.SortDir = spec.DefaultDir
		}
	}
	if q.SortDir == "" {
		q.SortDir = "asc"
	}
	return l.Query(ctx, q)
}

// check refuses what the list does not know: an unknown wish is an error,
// never silently ignored.
func (s ListSpec) check(q ListQuery) error {
	for _, f := range q.Filter {
		if !s.knowsFilter(f) {
			return refuse("unknown filter %q; known: %s", f, strings.Join(s.knownFilters(), ", "))
		}
	}
	switch {
	case q.SortBy != "" && !slices.Contains(s.Sorts, q.SortBy):
		return refuse("unknown sortBy %q; known: %s", q.SortBy, strings.Join(s.Sorts, ", "))
	case q.SortDir != "" && q.SortDir != "asc" && q.SortDir != "desc":
		return refuse("unknown sortDir %q; known: asc, desc", q.SortDir)
	case q.Limit < 0 || q.Offset < 0:
		return refuse("limit and offset cannot be negative")
	}
	return nil
}

func (s ListSpec) knowsFilter(f string) bool {
	if slices.Contains(s.Filters, f) {
		return true
	}
	for _, family := range s.FilterFamilies {
		if strings.HasPrefix(f, family) && len(f) > len(family) {
			return true
		}
	}
	return false
}

func (s ListSpec) knownFilters() []string {
	known := slices.Clone(s.Filters)
	for _, family := range s.FilterFamilies {
		known = append(known, family+"<id>")
	}
	if len(known) == 0 {
		return []string{"none"}
	}
	return known
}

// itemRules say how a list held in memory answers a query: what search looks
// in, what each filter keeps, what each order compares.
type itemRules[T any] struct {
	search func(T) []string
	// filter answers for every filter of the spec, its families included.
	filter func(name string) func(T) bool
	sorts  map[string]func(a, b T) int
}

// pageItems answers a query from the items the gates let through: for a
// list whose gated set is small (one organisation's members, one workshop's
// participants, one caller's keys). Filters combine; search is
// case-insensitive and partial; ties keep the gated query's order.
func pageItems[T any](items []T, q ListQuery, r itemRules[T]) Page[T] {
	shown := slices.Clone(items)
	for _, name := range q.Filter {
		shown = slices.DeleteFunc(shown, func(t T) bool { return !r.filter(name)(t) })
	}
	if needle := strings.ToLower(strings.TrimSpace(q.Search)); needle != "" {
		shown = slices.DeleteFunc(shown, func(t T) bool {
			return !slices.ContainsFunc(r.search(t), func(text string) bool {
				return strings.Contains(strings.ToLower(text), needle)
			})
		})
	}
	if compare := r.sorts[q.SortBy]; compare != nil {
		slices.SortStableFunc(shown, func(a, b T) int {
			if q.SortDir == "desc" {
				return compare(b, a)
			}
			return compare(a, b)
		})
	}
	total := len(shown)
	start := min(q.Offset, total)
	end := total
	if q.Limit > 0 {
		end = min(start+q.Limit, total)
	}
	return Page[T]{Items: shown[start:end], Total: total}
}

// byText compares texts case-insensitively; empty texts go last.
func byText[T any](text func(T) string) func(a, b T) int {
	return func(a, b T) int {
		ta, tb := strings.ToLower(text(a)), strings.ToLower(text(b))
		switch {
		case ta == tb:
			return 0
		case ta == "":
			return 1
		case tb == "":
			return -1
		}
		return strings.Compare(ta, tb)
	}
}

// byValue compares ordered values.
func byValue[T any, V cmp.Ordered](value func(T) V) func(a, b T) int {
	return func(a, b T) int { return cmp.Compare(value(a), value(b)) }
}
