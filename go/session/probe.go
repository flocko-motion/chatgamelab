package session

import (
	"context"
	"errors"
	"fmt"

	"cgl/core"
	"cgl/log"

	"engine"

	"github.com/google/uuid"
)

// ProbeResult says whether a model answered a probe, and what the probe cost.
type ProbeResult struct {
	Answers bool
	// Problem is the engine's problem when it did not answer ("failed",
	// "key_rejected", …), or "no_probe" for a model type without a probe.
	Problem string
	Message string
	Cost    float64
	Priced  bool
}

// ProbeModel asks a model of a platform whether it answers, for the admin
// (core.MayProbeModels), with the smallest request its type takes. It pays
// as the editor does: the admin's own payers on that platform, no game's
// sponsor. Without one it is refused (400 no_api_key); a model that does not
// answer is a result, not an error.
func ProbeModel(ctx context.Context, platform engine.Platform, kind engine.ModelType, model string) (ProbeResult, error) {
	if err := core.MayProbeModels(ctx); err != nil {
		return ProbeResult{}, err
	}
	payers, steps, err := core.OwnPayers(ctx)
	if err != nil {
		return ProbeResult{}, err
	}
	traceKeys(uuid.Nil, "", steps)
	offers, _ := core.Offers(payers, func(o core.PayerOffer) error {
		if o.Offer.Platform != platform {
			return engine.Refuse(engine.ProblemNoKey, fmt.Sprintf("the key is on %s", o.Offer.Platform))
		}
		return nil
	})
	if len(offers) == 0 {
		return ProbeResult{}, aiRefusal("the probe", engine.Refuse(engine.ProblemNoKey, fmt.Sprintf("no API key of yours is on %s", platform)))
	}
	chosen := offers[0]

	usage, err := engine.ProbeModel(ctx, chosen.Offer, kind, model)
	result := ProbeResult{Answers: err == nil, Cost: usage.TotalCost, Priced: usage.Complete}
	var refusal *engine.Refusal
	switch {
	case err == nil:
	case errors.Is(err, engine.ErrNoProbe):
		result.Problem, result.Message = "no_probe", err.Error()
	case errors.As(err, &refusal):
		result.Problem, result.Message = string(refusal.Problem), refusal.Error()
		if refusal.Problem == engine.ProblemKeyRejected {
			chosen.Payer.RecordCheck(ctx, false, string(engine.ProblemKeyRejected))
		}
	default:
		result.Problem, result.Message = string(engine.ProblemFailed), err.Error()
	}
	log.Info("model probed", "platform", platform, "type", kind, "model", model, "source", chosen.Payer.Source,
		"answers", result.Answers, "problem", result.Problem, "cost", result.Cost)
	return result, nil
}
