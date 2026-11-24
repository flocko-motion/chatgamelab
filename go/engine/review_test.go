package engine

import (
	"context"
	"testing"
)

func fullForm() ReviewSpec {
	return ReviewSpec{
		Genre: GenreAdventure, Title: "Lighthouse", Description: "Keep the light burning through the storm.",
		Scenario: "The player keeps a lighthouse; the goal is to guide three ships home.", GameStart: "A storm rises.",
		ImageStyle: "watercolour",
	}
}

// An empty required field is the point, found without asking a model.
func TestAReviewNamesAMissingFieldWithoutACall(t *testing.T) {
	spec := fullForm()
	spec.Scenario = "  "
	review, err := ReviewGame(context.Background(), spec, MockOffer())
	if err != nil {
		t.Fatalf("ReviewGame: %v", err)
	}
	if review.Field != "scenario" || !review.Missing {
		t.Errorf("got %+v, want the missing scenario", review)
	}
	if len(review.Usage.ByNode) != 0 {
		t.Errorf("a missing field was billed: %+v", review.Usage)
	}
}

func TestAReviewOfAFullFormAsksTheModelForOnePoint(t *testing.T) {
	review, err := ReviewGame(context.Background(), fullForm(), MockOffer())
	if err != nil {
		t.Fatalf("ReviewGame: %v", err)
	}
	// The mock answers the first choice, which is nothing to fix.
	if review.Field != "" || review.Missing || review.Text == "" {
		t.Errorf("got %+v, want nothing to fix, said in a sentence", review)
	}
	if len(review.Usage.ByNode) != 1 || review.Usage.ByNode[0].Node != reviewNode {
		t.Errorf("billed %+v, want the one review call", review.Usage.ByNode)
	}
}
