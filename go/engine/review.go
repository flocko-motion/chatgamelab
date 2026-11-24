package engine

import (
	"context"
	"encoding/json"
	"fmt"
	"strings"

	"engine/internal/adapters"
	"engine/internal/genre"
	"engine/internal/ports"
)

// ReviewSpec is a game's form as its author fills it in, for a review: the
// genre, the fields, and the language the answer is written in.
type ReviewSpec struct {
	Genre                                                                         Genre
	Title, Description, Scenario, GameStart, ImageStyle, VoiceStyle, StatusFields string
	Language                                                                      string
	MockPaceMs                                                                    int
}

// Review is the one point a review makes: the field it is about, empty when
// there is nothing to fix, and one sentence in the spec's language. Missing
// says a required field is empty, which needs no sentence: the client says
// it in its own words.
type Review struct {
	Field   string
	Missing bool
	Text    string
	Usage   UsageReport
}

// Missing is the first required field of the form left empty, "" when
// none is: a review names it without asking anyone.
func (spec ReviewSpec) Missing() string {
	required := []struct{ name, value string }{
		{"title", spec.Title}, {"scenario", spec.Scenario}, {"gameStart", spec.GameStart}, {"imageStyle", spec.ImageStyle},
	}
	for _, f := range required {
		if strings.TrimSpace(f.value) == "" {
			return f.name
		}
	}
	return ""
}

// reviewNode is the name a review is billed under.
const reviewNode = "review"

// ReviewGame finds the one point most worth improving in a game's form. A
// required field left empty is that point, found without a call; otherwise
// the offer's cheapest tooling model is asked, once, for a field done very
// badly, or for nothing. It blocks until the answer is in.
func ReviewGame(ctx context.Context, spec ReviewSpec, offer Offer) (Review, error) {
	if field := spec.Missing(); field != "" {
		return Review{Field: field, Missing: true, Usage: newUsageLedger().report()}, nil
	}

	for _, issue := range offerIssues("offer", offer) {
		if issue.Severity == SeverityError {
			return Review{}, fmt.Errorf("%s", issue)
		}
	}
	tiers := offer.Caveats.tiers()
	if len(tiers) == 0 {
		return Review{}, Refuse(ProblemNoKey, fmt.Sprintf("offer %q has no tier", offer.ID))
	}
	toggles, _ := offer.Caveats.allowed(tiers[0])
	kit, err := SessionSpec{MockPaceMs: spec.MockPaceMs}.adapters(settlement{offer: offer, Choice: Choice{Tier: tiers[0], Toggles: toggles}})
	if err != nil {
		return Review{}, err
	}
	defer kit.close()
	tool, ok := kit.tool.(adapters.SchemaTool)
	if !ok {
		return Review{}, Refuse(ProblemNoKey, fmt.Sprintf("platform %q cannot review", offer.Platform))
	}

	ledger := newUsageLedger()
	ledger.price(offer.Platform)
	system, message, schema := genre.Review(string(spec.Genre), genre.ReviewForm{
		Title: spec.Title, Description: spec.Description, Scenario: spec.Scenario, GameStart: spec.GameStart,
		ImageStyle: spec.ImageStyle, VoiceStyle: spec.VoiceStyle, StatusFields: spec.StatusFields,
	}, spec.Language)
	reply, used, err := tool.QuerySchema(ctx, system, message, *schema)
	ledger.record(ports.Usage{
		Node: reviewNode, Model: used.Model,
		InputTokens: used.InputTokens, CachedInputTokens: used.CachedInputTokens, OutputTokens: used.OutputTokens,
	})
	if err != nil {
		return Review{Usage: ledger.report()}, coverFailure("reviewing the game", err)
	}
	var point struct {
		Field string `json:"field"`
		Text  string `json:"text"`
	}
	if err := json.Unmarshal([]byte(reply), &point); err != nil {
		return Review{Usage: ledger.report()}, coverFailure("reviewing the game", fmt.Errorf("%w: %v", adapters.ErrBadReply, err))
	}
	if point.Field == "none" {
		point.Field = ""
	}
	return Review{Field: point.Field, Text: strings.TrimSpace(point.Text), Usage: ledger.report()}, nil
}
