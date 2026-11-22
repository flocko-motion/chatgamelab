// package: api / schemaf endpoints
// type:    contract
// job:     the query every list endpoint takes
// limits:  what a list knows and refuses is the core's (core.List)
package api

import "cgl/core"

// SortDir is the direction of a list's order.
type SortDir string

const (
	SortAsc  SortDir = "asc"
	SortDesc SortDir = "desc"
)

// ListParams is the query every list endpoint takes, embedded in its
// request: filters F (repeat to combine), search, order S and direction,
// page size and offset. All optional; a list refuses a filter or order it
// does not know, naming the known ones.
type ListParams[F ~string, S ~string] struct {
	Filter  []F      `query:"filter"`
	Search  *string  `query:"search"`
	SortBy  *S       `query:"sortBy"`
	SortDir *SortDir `query:"sortDir"`
	Limit   *int     `query:"limit"`
	Offset  *int     `query:"offset"`
}

// query is the parameters as the core's list query.
func (p ListParams[F, S]) query() core.ListQuery {
	q := core.ListQuery{Search: opt(p.Search)}
	for _, f := range p.Filter {
		q.Filter = append(q.Filter, string(f))
	}
	if p.SortBy != nil {
		q.SortBy = string(*p.SortBy)
	}
	if p.SortDir != nil {
		q.SortDir = string(*p.SortDir)
	}
	if p.Limit != nil {
		q.Limit = *p.Limit
	}
	if p.Offset != nil {
		q.Offset = *p.Offset
	}
	return q
}

// NoFilter is the filter type of a list that offers none: any filter is refused.
type NoFilter string

// KeyFilter narrows a key list to one platform: "platform:<id>", one per
// platform; several combine.
type KeyFilter string
