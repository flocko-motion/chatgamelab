package wire

import "cgl/core"

// Page is one page of a list, and how many items match across all pages.
// Every list endpoint answers with one (schema `<Item>Page`).
type Page[T any] struct {
	Items []T `json:"items"`
	Total int `json:"total"`
}

// pageOf converts a core page item by item, passing a loading error through.
func pageOf[C, V any](p core.Page[C], err error, view func(C) V) (Page[V], error) {
	if err != nil {
		return Page[V]{}, err
	}
	page := Page[V]{Items: make([]V, len(p.Items)), Total: p.Total}
	for i, item := range p.Items {
		page.Items[i] = view(item)
	}
	return page, nil
}
