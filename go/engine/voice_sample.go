package engine

import (
	"bytes"
	"context"
	"encoding/binary"
	"encoding/json"
	"fmt"
	"strings"

	"engine/internal/adapters"
	"engine/internal/adapters/openai"
	"engine/internal/blocks"
	"engine/internal/genre"
	"engine/internal/ports"
)

// VoiceSampleSpec is a voice to hear outside any game: the genre it is for,
// the author's voice style, the scenario as background for describing it,
// the line it says, and the language that line is in.
type VoiceSampleSpec struct {
	Genre      Genre
	VoiceStyle string
	Scenario   string
	Line       string
	Language   string
	// Renew drops the voice made for the voice style on the key, and its
	// sample, and makes a new one: for every game with that voice style on
	// that key.
	Renew      bool
	MockPaceMs int
}

// VoiceSample is a voice saying its line, as WAV, and what hearing it cost.
type VoiceSample struct {
	Audio       []byte
	ContentType string
	// Voice is the made voice's id; empty for a built-in voice.
	Voice string
	// Stored says the sample was kept from an earlier preview: no provider
	// was asked, and nothing was billed.
	Stored bool
	Usage  UsageReport
}

// The names a sample's calls are billed under.
const (
	sampleNodeBrief = "voice-brief"
	sampleNodeSay   = "voice-sample"
)

// SpeakSample has a voice say a line on one offer, as a game of the genre
// would speak: in the voice made for the voice style on the offer's key
// where the genre makes voices of its own (made now when there is none
// yet), else in a built-in voice the voice style steers. A made voice's
// sample is kept in the offer's voice store, when it keeps samples, and
// played from there on later previews without asking the provider; it
// keeps the language it was first made in. An offer that cannot speak live
// is refused with ProblemNoKey.
func SpeakSample(ctx context.Context, spec VoiceSampleSpec, offer Offer) (VoiceSample, error) {
	for _, issue := range offerIssues("offer", offer) {
		if issue.Severity == SeverityError {
			return VoiceSample{}, fmt.Errorf("%s", issue)
		}
	}
	tier, ok := LiveTier(offer)
	if !ok {
		return VoiceSample{}, Refuse(ProblemNoKey, fmt.Sprintf("offer %q speaks no live voice", offer.ID))
	}
	toggles, _ := offer.Caveats.allowed(tier)
	kit, err := SessionSpec{MockPaceMs: spec.MockPaceMs}.adapters(settlement{offer: offer, Choice: Choice{Tier: tier, Toggles: toggles}})
	if err != nil {
		return VoiceSample{}, err
	}
	defer kit.close()
	sayer, ok := kit.live.(adapters.Sayer)
	if !ok {
		return VoiceSample{}, Refuse(ProblemNoKey, fmt.Sprintf("platform %q speaks no sample", offer.Platform))
	}

	ledger := newUsageLedger()
	ledger.price(offer.Platform)
	bill := func(node string, used adapters.Usage) {
		ledger.record(ports.Usage{
			Node: node, Model: used.Model,
			InputTokens: used.InputTokens, CachedInputTokens: used.CachedInputTokens, OutputTokens: used.OutputTokens,
			AudioOutputTokens: used.AudioOutputTokens, AudioSeconds: used.AudioSeconds,
		})
	}

	style := strings.TrimSpace(spec.VoiceStyle)
	samples, _ := offer.Voices.(adapters.VoiceSamples)
	ownVoice := style != "" && spec.Genre.MakesVoices() && kit.voices != nil && samples != nil
	hash := blocks.StyleHash(style)

	voice := ""
	if ownVoice {
		if spec.Renew {
			if old, removed, err := samples.Remove(ctx, hash); err != nil {
				return VoiceSample{}, err
			} else if removed {
				// The provider's copy goes too; one it cannot delete is an
				// orphan the next voice made on the key clears away.
				_ = kit.voices.Delete(ctx, old)
			}
		} else if audio, contentType, kept, err := samples.Sample(ctx, hash); err == nil && kept {
			id, _, _ := samples.Lookup(ctx, hash)
			return VoiceSample{Audio: audio, ContentType: contentType, Voice: id, Stored: true, Usage: ledger.report()}, nil
		}

		description := style
		if kit.tool != nil {
			if tool, ok := kit.tool.(adapters.SchemaTool); ok {
				system, message, schema := genre.VoiceBrief(style, spec.Scenario, spec.Language, openai.LiveVoices)
				reply, used, err := tool.QuerySchema(ctx, system, message, *schema)
				bill(sampleNodeBrief, used)
				var brief struct {
					Description string `json:"description"`
				}
				if err == nil && json.Unmarshal([]byte(reply), &brief) == nil && strings.TrimSpace(brief.Description) != "" {
					description = strings.TrimSpace(brief.Description)
				}
			}
		}
		slot := &adapters.VoicesSlot{}
		slot.Set(kit.voices, samples)
		voice, err = blocks.NewVoice(sampleNodeSay, style, slot).Make(ctx, description)
		if err != nil {
			return VoiceSample{Usage: ledger.report()}, coverFailure("making the voice", err)
		}
	}

	pcm, used, err := sayer.Say(ctx, adapters.SayRequest{Voice: voice, Instructions: sayInstructions(style), Line: spec.Line})
	bill(sampleNodeSay, used)
	if err != nil {
		return VoiceSample{Usage: ledger.report()}, coverFailure("speaking the sample", err)
	}
	audio := wav(pcm, adapters.SayRate)
	if ownVoice {
		// Kept only while the voice is: a voice cleared away meanwhile keeps none.
		_ = samples.KeepSample(ctx, hash, audio, "audio/wav")
	}
	return VoiceSample{Audio: audio, ContentType: "audio/wav", Voice: voice, Usage: ledger.report()}, nil
}

// sayInstructions is how a sample is spoken: one line, as written, in the
// way the voice style says, as a game's voice follows it.
func sayInstructions(style string) string {
	text := "You read one line aloud, as a sample of your voice. Say it exactly as written, in full, and then stay silent."
	if style != "" {
		text += "\nYour voice: " + style
	}
	return text
}

// LiveTier is the cheapest tier of the offer that speaks live, whatever its
// default: a sample is a few seconds of one voice.
func LiveTier(offer Offer) (Tier, bool) {
	for _, tier := range offer.Caveats.tiers() {
		if toggles, _ := offer.Caveats.allowed(tier); toggles.AudioLive {
			return tier, true
		}
	}
	return "", false
}

// wav wraps 16-bit mono PCM in a WAV header, so a browser plays it as it is.
func wav(pcm []byte, rate int) []byte {
	var b bytes.Buffer
	le := func(v any) { _ = binary.Write(&b, binary.LittleEndian, v) }
	b.WriteString("RIFF")
	le(uint32(36 + len(pcm)))
	b.WriteString("WAVEfmt ")
	le(uint32(16))       // the format chunk's size
	le(uint16(1))        // PCM
	le(uint16(1))        // mono
	le(uint32(rate))     // samples a second
	le(uint32(rate * 2)) // bytes a second
	le(uint16(2))        // bytes a frame
	le(uint16(16))       // bits a sample
	b.WriteString("data")
	le(uint32(len(pcm)))
	b.Write(pcm)
	return b.Bytes()
}
