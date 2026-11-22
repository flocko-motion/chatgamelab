package core

import (
	"context"
	"strings"
	"testing"
)

type fruit struct {
	name  string
	color string
	size  int
}

var fruits = []fruit{
	{"banana", "yellow", 3},
	{"Apple", "red", 2},
	{"cherry", "red", 1},
	{"", "green", 4},
}

var fruitRules = itemRules[fruit]{
	search: func(f fruit) []string { return []string{f.name, f.color} },
	filter: func(name string) func(fruit) bool {
		color := strings.TrimPrefix(name, "color:")
		return func(f fruit) bool { return f.color == color }
	},
	sorts: map[string]func(a, b fruit) int{
		"name": byText(func(f fruit) string { return f.name }),
		"size": byValue(func(f fruit) int { return f.size }),
	},
}

// fruitList answers through pageItems and records the query it got.
type fruitList struct{ got *ListQuery }

func (fruitList) Spec() ListSpec {
	return ListSpec{FilterFamilies: []string{"color:"}, Sorts: []string{"name", "size"}, DefaultSort: "size", DefaultDir: "desc"}
}

func (l fruitList) Query(_ context.Context, q ListQuery) (Page[fruit], error) {
	*l.got = q
	return pageItems(fruits, q, fruitRules), nil
}

func names(p Page[fruit]) string {
	var out []string
	for _, f := range p.Items {
		out = append(out, f.name)
	}
	return strings.Join(out, ",")
}

func TestPageItems(t *testing.T) {
	cases := []struct {
		name  string
		q     ListQuery
		want  string
		total int
	}{
		{"by name, empty last", ListQuery{SortBy: "name", SortDir: "asc"}, "Apple,banana,cherry,", 4},
		{"by size descending", ListQuery{SortBy: "size", SortDir: "desc"}, ",banana,Apple,cherry", 4},
		{"filter", ListQuery{Filter: []string{"color:red"}, SortBy: "name", SortDir: "asc"}, "Apple,cherry", 2},
		{"search any case, any text", ListQuery{Search: "YEL", SortBy: "name", SortDir: "asc"}, "banana", 1},
		{"page", ListQuery{SortBy: "size", SortDir: "asc", Limit: 2, Offset: 1}, "Apple,banana", 4},
		{"page past the end", ListQuery{SortBy: "size", SortDir: "asc", Limit: 2, Offset: 9}, "", 4},
	}
	for _, c := range cases {
		t.Run(c.name, func(t *testing.T) {
			got := pageItems(fruits, c.q, fruitRules)
			if names(got) != c.want || got.Total != c.total {
				t.Errorf("got %q (total %d), want %q (total %d)", names(got), got.Total, c.want, c.total)
			}
		})
	}
}

func TestListAppliesTheDefaultOrder(t *testing.T) {
	var got ListQuery
	if _, err := List(context.Background(), fruitList{&got}, ListQuery{}); err != nil {
		t.Fatal(err)
	}
	if got.SortBy != "size" || got.SortDir != "desc" {
		t.Errorf("got order %s %s, want size desc", got.SortBy, got.SortDir)
	}
	if _, err := List(context.Background(), fruitList{&got}, ListQuery{SortBy: "name"}); err != nil {
		t.Fatal(err)
	}
	if got.SortBy != "name" || got.SortDir != "asc" {
		t.Errorf("got order %s %s, want name asc", got.SortBy, got.SortDir)
	}
}

func TestListRefusesWhatItDoesNotKnow(t *testing.T) {
	var got ListQuery
	for _, c := range []struct {
		q    ListQuery
		want string
	}{
		{ListQuery{Filter: []string{"ripe"}}, `unknown filter "ripe"; known: color:<id>`},
		{ListQuery{Filter: []string{"color:"}}, `unknown filter "color:"`},
		{ListQuery{SortBy: "weight"}, `unknown sortBy "weight"; known: name, size`},
		{ListQuery{SortDir: "up"}, `unknown sortDir "up"`},
		{ListQuery{Limit: -1}, "cannot be negative"},
	} {
		_, err := List(context.Background(), fruitList{&got}, c.q)
		if err == nil || !strings.Contains(err.Error(), c.want) {
			t.Errorf("%+v: got %v, want an error containing %q", c.q, err, c.want)
		}
	}
}
