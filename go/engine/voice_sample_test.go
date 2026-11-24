package engine

import (
	"bytes"
	"context"
	"testing"

	"engine/internal/blocks"
)

// sampleOffer is the mock offer with a voice store of its own, as the
// platform hands one in for the payer's key.
func sampleOffer() (Offer, *MemoryVoices) {
	offer := MockOffer()
	store := &MemoryVoices{}
	offer.Voices = store
	return offer, store
}

func TestTheFirstSampleMakesTheVoiceAndKeepsWhatItSaid(t *testing.T) {
	offer, store := sampleOffer()
	spec := VoiceSampleSpec{Genre: GenreNPC, VoiceStyle: "deep and slow", Line: "Hello there."}

	first, err := SpeakSample(context.Background(), spec, offer)
	if err != nil {
		t.Fatalf("SpeakSample: %v", err)
	}
	if first.Stored {
		t.Error("the first sample came from the store, but none was kept yet")
	}
	if first.ContentType != "audio/wav" || !bytes.HasPrefix(first.Audio, []byte("RIFF")) {
		t.Errorf("the sample is no WAV: %q, %d bytes", first.ContentType, len(first.Audio))
	}
	if first.Voice == "" {
		t.Error("the sample names no voice; a voice of its own was to be made")
	}
	if ids, _ := store.IDs(context.Background()); len(ids) != 1 {
		t.Errorf("the store keeps %d voices, want the one made", len(ids))
	}

	again, err := SpeakSample(context.Background(), spec, offer)
	if err != nil {
		t.Fatalf("SpeakSample again: %v", err)
	}
	if !again.Stored || !bytes.Equal(again.Audio, first.Audio) {
		t.Error("the second sample was spoken again; it was to come from the store")
	}
	if again.Usage.TotalCost != 0 || len(again.Usage.ByNode) != 0 {
		t.Errorf("a stored sample was billed: %+v", again.Usage)
	}
}

func TestANewVoiceReplacesTheVoiceAndItsSample(t *testing.T) {
	offer, store := sampleOffer()
	spec := VoiceSampleSpec{Genre: GenreNPC, VoiceStyle: "bright and quick", Line: "Hello there."}
	first, err := SpeakSample(context.Background(), spec, offer)
	if err != nil {
		t.Fatalf("SpeakSample: %v", err)
	}

	spec.Renew = true
	renewed, err := SpeakSample(context.Background(), spec, offer)
	if err != nil {
		t.Fatalf("SpeakSample renew: %v", err)
	}
	if renewed.Stored {
		t.Error("a new voice played the old sample")
	}
	if ids, _ := store.IDs(context.Background()); len(ids) != 1 {
		t.Errorf("the store keeps %d voices after renewing, want only the new one", len(ids))
	}
	if _, _, kept, _ := store.Sample(context.Background(), blocks.StyleHash(spec.VoiceStyle)); !kept {
		t.Error("the new voice's sample was not kept")
	}
	_ = first
}

func TestWithoutVoiceStyleTheBuiltInVoiceSpeaksAndNothingIsKept(t *testing.T) {
	offer, store := sampleOffer()
	sample, err := SpeakSample(context.Background(), VoiceSampleSpec{Genre: GenreNPC, Line: "Hello there."}, offer)
	if err != nil {
		t.Fatalf("SpeakSample: %v", err)
	}
	if sample.Stored || len(sample.Audio) == 0 {
		t.Errorf("stored %v, %d bytes; want a fresh sample", sample.Stored, len(sample.Audio))
	}
	if ids, _ := store.IDs(context.Background()); len(ids) != 0 {
		t.Errorf("a voice was made without a voice style: %v", ids)
	}
}

// A genre that makes no voice of its own previews the built-in voice the
// voice style steers, and renewing makes nothing.
func TestAGenreWithoutVoicesOfItsOwnPreviewsTheBuiltInVoice(t *testing.T) {
	offer, store := sampleOffer()
	spec := VoiceSampleSpec{Genre: GenreAdventure, VoiceStyle: "warm narrator", Line: "Once upon a time.", Renew: true}
	sample, err := SpeakSample(context.Background(), spec, offer)
	if err != nil {
		t.Fatalf("SpeakSample: %v", err)
	}
	if sample.Stored || len(sample.Audio) == 0 {
		t.Errorf("stored %v, %d bytes; want a fresh sample", sample.Stored, len(sample.Audio))
	}
	if ids, _ := store.IDs(context.Background()); len(ids) != 0 {
		t.Errorf("a voice was made for a genre that makes none: %v", ids)
	}
}
