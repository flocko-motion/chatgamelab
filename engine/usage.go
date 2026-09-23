package engine

import (
	"sort"
	"sync"

	"engine/internal/adapters"
	"engine/internal/adapters/mock"
	"engine/internal/adapters/openai"
	"engine/internal/ports"
)

// UsageRecord is what one block spent on one model, cumulatively. Units differ
// because providers bill differently: text per token, live audio per minute,
// and pictures per picture.
type UsageRecord struct {
	Node              string  `json:"node,omitempty"`
	Model             string  `json:"model"`
	InputTokens       int64   `json:"inputTokens,omitempty"`
	CachedInputTokens int64   `json:"cachedInputTokens,omitempty"`
	OutputTokens      int64   `json:"outputTokens,omitempty"`
	AudioSeconds      float64 `json:"audioSeconds,omitempty"`
	Images            int64   `json:"images,omitempty"`
	// Cost is an estimate from a hand-maintained price table, because providers
	// do not publish prices through their APIs. Zero when the model is unpriced,
	// which Priced distinguishes from genuinely free.
	Cost   float64 `json:"cost"`
	Priced bool    `json:"priced"`
}

// UsageReport is the whole session's spending, both ways round: per block for a
// graph view's labels, and per model because that is what prices attach to.
type UsageReport struct {
	ByNode    []UsageRecord `json:"byNode"`
	ByModel   []UsageRecord `json:"byModel"`
	TotalCost float64       `json:"totalCost"`
	// Complete is false when any model in the session has no price, so a reader
	// knows the total is a floor rather than a figure.
	Complete bool `json:"complete"`
}

// usageLedger keeps the latest total per block and model. Blocks report
// cumulative totals, so a report is a replacement rather than an addition and a
// dropped event costs accuracy for an instant rather than permanently.
type usageLedger struct {
	mu     sync.Mutex
	latest map[usageKey]ports.Usage
	prices func(string) (adapters.Price, bool)
}

type usageKey struct{ node, model string }

func newUsageLedger(platform Platform) *usageLedger {
	lookup := mock.Prices
	if platform == PlatformOpenAI {
		lookup = openai.Prices
	}
	return &usageLedger{latest: map[usageKey]ports.Usage{}, prices: lookup}
}

func (l *usageLedger) record(u ports.Usage) {
	l.mu.Lock()
	defer l.mu.Unlock()
	l.latest[usageKey{node: u.Node, model: u.Model}] = u
}

func (l *usageLedger) report() UsageReport {
	l.mu.Lock()
	defer l.mu.Unlock()

	report := UsageReport{Complete: true}
	byModel := map[string]*UsageRecord{}

	for key, used := range l.latest {
		record := l.priced(key.node, key.model, used)
		report.ByNode = append(report.ByNode, record)

		total, seen := byModel[key.model]
		if !seen {
			total = &UsageRecord{Model: key.model}
			byModel[key.model] = total
		}
		total.InputTokens += record.InputTokens
		total.CachedInputTokens += record.CachedInputTokens
		total.OutputTokens += record.OutputTokens
		total.AudioSeconds += record.AudioSeconds
		total.Images += record.Images
		total.Cost += record.Cost
		total.Priced = record.Priced

		report.TotalCost += record.Cost
		if !record.Priced {
			report.Complete = false
		}
	}

	for _, total := range byModel {
		report.ByModel = append(report.ByModel, *total)
	}

	sort.Slice(report.ByNode, func(i, j int) bool { return report.ByNode[i].Node < report.ByNode[j].Node })
	sort.Slice(report.ByModel, func(i, j int) bool { return report.ByModel[i].Model < report.ByModel[j].Model })
	return report
}

func (l *usageLedger) priced(node, model string, used ports.Usage) UsageRecord {
	record := UsageRecord{
		Node:              node,
		Model:             model,
		InputTokens:       used.InputTokens,
		CachedInputTokens: used.CachedInputTokens,
		OutputTokens:      used.OutputTokens,
		AudioSeconds:      used.AudioSeconds,
		Images:            used.Images,
	}

	price, known := l.prices(model)
	if !known {
		return record
	}
	record.Priced = true
	record.Cost = price.Cost(adapters.Usage{
		Model:             model,
		InputTokens:       used.InputTokens,
		CachedInputTokens: used.CachedInputTokens,
		OutputTokens:      used.OutputTokens,
		AudioSeconds:      used.AudioSeconds,
		Images:            used.Images,
	})
	return record
}
