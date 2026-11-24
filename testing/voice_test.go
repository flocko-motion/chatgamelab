package testing

import (
	"strings"
	"testing"
	"time"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// VoiceTestSuite tests that a game's voice is made once per voice style and
// key, and reused by every later session on that key, a copied game's too.
type VoiceTestSuite struct {
	testutil.BaseSuite
}

func TestVoiceSuite(t *testing.T) {
	s := &VoiceTestSuite{}
	s.SuiteName = "Voice Tests"
	suite.Run(t, s)
}

// voiceWords are what the voice block says it did, beside the phases every
// block reports.
var voiceWords = map[string]bool{"created": true, "reused": true, "built-in": true, "create-failed": true, "timeout": true, "store-failed": true}

// voiceOutcome waits for the voice block's word in a session's journal:
// created, reused, built-in, …
func (s *VoiceTestSuite) voiceOutcome(user *testutil.UserClient, sessionID string) string {
	deadline := time.Now().Add(10 * time.Second)
	for time.Now().Before(deadline) {
		for _, e := range Must(user.SessionJournal(sessionID)) {
			if e.Source == "voice" && voiceWords[e.Kind] {
				return e.Kind
			}
		}
		time.Sleep(100 * time.Millisecond)
	}
	s.FailNow("the voice block said nothing")
	return ""
}

// The first session on a key makes the game's voice; the next one, and a
// copy of the game with the same voice style, reuse it; another key makes
// its own.
func (s *VoiceTestSuite) TestAVoiceIsMadeOncePerKeyAndStyle() {
	alice := s.CreateUser("voice-alice")
	Must(alice.AddApiKey("mock-voice-alice", "Alice's key", "mock"))
	game := Must(alice.UploadGame("bridge-troll"))
	game.VoiceStyle = "deep, slow and a little hoarse"
	Must(alice.UpdateGame(game.Id.String(), game))

	first := Must(alice.CreateSession(testutil.ForGame(game.Id.String())))
	s.Equal("created", s.voiceOutcome(alice, first), "the first session makes the voice")

	again := Must(alice.CreateSession(testutil.ForGame(game.Id.String())))
	s.Equal("reused", s.voiceOutcome(alice, again), "a later session reuses it")

	copied := Must(alice.UploadGame("bridge-troll"))
	copied.VoiceStyle = game.VoiceStyle
	Must(alice.UpdateGame(copied.Id.String(), copied))
	other := Must(alice.CreateSession(testutil.ForGame(copied.Id.String())))
	s.Equal("reused", s.voiceOutcome(alice, other), "another game with the same voice style reuses it")

	bob := s.CreateUser("voice-bob")
	Must(bob.AddApiKey("mock-voice-bob", "Bob's key", "mock"))
	public := Must(alice.SetGamePublic(game.Id.String(), true))
	bobs := Must(bob.CreateSession(testutil.ForGame(public.Id.String())))
	s.Equal("created", s.voiceOutcome(bob, bobs), "another key makes its own")
}

// A game without a voice style speaks in the built-in voice.
func (s *VoiceTestSuite) TestNoVoiceStyleKeepsTheBuiltInVoice() {
	carol := s.CreateUser("voice-carol")
	Must(carol.AddApiKey("mock-voice-carol", "Carol's key", "mock"))
	game := Must(carol.UploadGame("bridge-troll"))
	game.VoiceStyle = ""
	Must(carol.UpdateGame(game.Id.String(), game))

	id := Must(carol.CreateSession(testutil.ForGame(game.Id.String())))
	s.True(strings.HasPrefix(s.voiceOutcome(carol, id), "built-in"))
}

// The editor hears a voice: the first sample makes the voice and keeps what
// it said, the next one plays it from the store, and a new voice replaces
// both. The session that follows speaks in the voice the preview made.
func (s *VoiceTestSuite) TestAPreviewKeepsItsSampleUntilANewVoiceIsMade() {
	dana := s.CreateUser("voice-dana")
	Must(dana.AddApiKey("mock-voice-dana", "Dana's key", "mock"))
	style := "warm, low and unhurried"

	first := Must(dana.VoiceSample("npc-live", style, "A lighthouse keeper."))
	s.False(first.Stored, "the first sample is spoken")
	s.True(first.OwnVoice, "a voice of its own speaks")
	s.Equal("audio/wav", first.ContentType)
	s.True(strings.HasPrefix(string(first.Audio), "RIFF"), "the sample is a WAV")

	again := Must(dana.VoiceSample("npc-live", style, "A lighthouse keeper."))
	s.True(again.Stored, "the second sample comes from the store, without the provider")
	s.Equal(first.Audio, again.Audio)

	renewed := Must(dana.NewVoice("npc-live", style, "A lighthouse keeper."))
	s.False(renewed.Stored, "a new voice drops the old sample")
	s.True(Must(dana.VoiceSample("npc-live", style, "")).Stored, "the new voice's sample is kept")

	game := Must(dana.UploadGame("bridge-troll"))
	game.Genre = "npc-live"
	game.VoiceStyle = style
	Must(dana.UpdateGame(game.Id.String(), game))
	id := Must(dana.CreateSession(testutil.ForGame(game.Id.String())))
	s.Equal("reused", s.voiceOutcome(dana, id), "a game with the style speaks in the voice the preview made")
}

// A genre that makes no voice of its own, and an empty voice style, hear a
// built-in voice; nothing is kept.
func (s *VoiceTestSuite) TestWithoutAVoiceOfItsOwnTheBuiltInVoiceIsHeard() {
	erin := s.CreateUser("voice-erin")
	Must(erin.AddApiKey("mock-voice-erin", "Erin's key", "mock"))

	narrated := Must(erin.VoiceSample("adventure", "a warm narrator", ""))
	s.False(narrated.OwnVoice)
	s.False(Must(erin.VoiceSample("adventure", "a warm narrator", "")).Stored, "a built-in voice's sample is not kept")
	s.False(Must(erin.VoiceSample("npc-live", "", "")).OwnVoice, "no voice style, no voice of its own")
}
