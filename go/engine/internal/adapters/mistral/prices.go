package mistral

import "engine/internal/adapters"

// prices is empty on purpose. The tiers name "-latest" aliases, which Mistral
// moves to new model versions at new prices, so any figure here would go stale
// without notice. An unpriced model shows as unpriced in the usage report.
var prices = map[string]adapters.Price{}

// Prices reports whether a model is priced, so a caller can say "unpriced" over a zero.
func Prices(model string) (adapters.Price, bool) {
	price, ok := prices[model]
	return price, ok
}
