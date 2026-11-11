package obj

import "testing"

func TestCapAiQualityTier(t *testing.T) {
	cases := []struct {
		requested, limit, applied string
		capped                    bool
	}{
		{AiModelMax, AiModelBalanced, AiModelBalanced, true},
		{AiModelPremium, AiModelBalanced, AiModelBalanced, true},
		{AiModelBalanced, AiModelBalanced, AiModelBalanced, false},
		{AiModelEconomy, AiModelBalanced, AiModelEconomy, false},
		{AiModelMax, AiModelMax, AiModelMax, false},
		{AiModelMax, "", AiModelMax, false},
		{AiModelMax, "unknown", AiModelMax, false},
	}
	for _, c := range cases {
		applied, capped := CapAiQualityTier(c.requested, c.limit)
		if applied != c.applied || capped != c.capped {
			t.Errorf("CapAiQualityTier(%q, %q) = %q, %v; want %q, %v", c.requested, c.limit, applied, capped, c.applied, c.capped)
		}
	}
}
