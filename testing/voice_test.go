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

// custom is the character's voice set to a voice of the game's own.
var custom = &map[string]string{"character": "custom"}

// The first session on a key makes the game's voice; the next one, and a
// copy of the game with the same voice style, reuse it; another key makes
// its own.
func (s *VoiceTestSuite) TestAVoiceIsMadeOncePerKeyAndStyle() {
	alice := s.CreateUser("voice-alice")
	Must(alice.AddApiKey("mock-voice-alice", "Alice's key", "mock"))
	game := Must(alice.UploadGame("bridge-troll"))
	game.VoiceStyle = "deep, slow and a little hoarse"
	game.Voices = custom
	Must(alice.UpdateGame(game.Id.String(), game))

	first := Must(alice.CreateSession(testutil.ForGame(game.Id.String()).InMode("live")))
	s.Equal("created", s.voiceOutcome(alice, first), "the first session makes the voice")

	again := Must(alice.CreateSession(testutil.ForGame(game.Id.String()).InMode("live")))
	s.Equal("reused", s.voiceOutcome(alice, again), "a later session reuses it")

	copied := Must(alice.UploadGame("bridge-troll"))
	copied.VoiceStyle = game.VoiceStyle
	copied.Voices = custom
	Must(alice.UpdateGame(copied.Id.String(), copied))
	other := Must(alice.CreateSession(testutil.ForGame(copied.Id.String()).InMode("live")))
	s.Equal("reused", s.voiceOutcome(alice, other), "another game with the same voice style reuses it")

	bob := s.CreateUser("voice-bob")
	Must(bob.AddApiKey("mock-voice-bob", "Bob's key", "mock"))
	public := Must(alice.SetGamePublic(game.Id.String(), true))
	bobs := Must(bob.CreateSession(testutil.ForGame(public.Id.String()).InMode("live")))
	s.Equal("created", s.voiceOutcome(bob, bobs), "another key makes its own")
}

// A character set to a built-in voice speaks in one, voice style or not:
// no voice is made for it.
func (s *VoiceTestSuite) TestABuiltInChoiceMakesNoVoice() {
	frank := s.CreateUser("voice-frank")
	Must(frank.AddApiKey("mock-voice-frank", "Frank's key", "mock"))
	game := Must(frank.UploadGame("bridge-troll"))
	game.VoiceStyle = "deep, slow and a little hoarse"
	game.Voices = &map[string]string{"character": "female"}
	Must(frank.UpdateGame(game.Id.String(), game))

	id := Must(frank.CreateSession(testutil.ForGame(game.Id.String()).InMode("live")))
	s.Equal("built-in", s.voiceOutcome(frank, id))
}

// A game without a voice style speaks in the built-in voice.
func (s *VoiceTestSuite) TestNoVoiceStyleKeepsTheBuiltInVoice() {
	carol := s.CreateUser("voice-carol")
	Must(carol.AddApiKey("mock-voice-carol", "Carol's key", "mock"))
	game := Must(carol.UploadGame("bridge-troll"))
	game.VoiceStyle = ""
	Must(carol.UpdateGame(game.Id.String(), game))

	id := Must(carol.CreateSession(testutil.ForGame(game.Id.String()).InMode("live")))
	s.True(strings.HasPrefix(s.voiceOutcome(carol, id), "built-in"))
}

// The editor hears a voice: the first sample makes the voice and keeps what
// it said, the next one plays it from the store, and a new voice replaces
// both. The session that follows speaks in the voice the preview made.
func (s *VoiceTestSuite) TestAPreviewKeepsItsSampleUntilANewVoiceIsMade() {
	dana := s.CreateUser("voice-dana")
	Must(dana.AddApiKey("mock-voice-dana", "Dana's key", "mock"))
	style := "warm, low and unhurried"

	first := Must(dana.VoiceSample("npc", style, "A lighthouse keeper."))
	s.False(first.Stored, "the first sample is spoken")
	s.True(first.OwnVoice, "a voice of its own speaks")
	s.Equal("audio/wav", first.ContentType)
	s.True(strings.HasPrefix(string(first.Audio), "RIFF"), "the sample is a WAV")

	again := Must(dana.VoiceSample("npc", style, "A lighthouse keeper."))
	s.True(again.Stored, "the second sample comes from the store, without the provider")
	s.Equal(first.Audio, again.Audio)

	renewed := Must(dana.NewVoice("npc", style, "A lighthouse keeper."))
	s.False(renewed.Stored, "a new voice drops the old sample")
	s.True(Must(dana.VoiceSample("npc", style, "")).Stored, "the new voice's sample is kept")

	game := Must(dana.UploadGame("bridge-troll"))
	game.Genre = "npc"
	game.VoiceStyle = style
	game.Voices = &map[string]string{"character": "custom"}
	Must(dana.UpdateGame(game.Id.String(), game))
	id := Must(dana.CreateSession(testutil.ForGame(game.Id.String()).InMode("live")))
	s.Equal("reused", s.voiceOutcome(dana, id), "a game with the style speaks in the voice the preview made")
}

// A voice set to a built-in choice is neither heard nor made in the editor,
// as the session casts its own; custom without a voice style hears a
// built-in voice, and nothing is kept.
func (s *VoiceTestSuite) TestOnlyACustomVoiceIsHeard() {
	erin := s.CreateUser("voice-erin")
	Must(erin.AddApiKey("mock-voice-erin", "Erin's key", "mock"))

	for _, voice := range []string{"male", "female", "any"} {
		_, err := erin.VoiceSampleFor(voice, "adventure", "a warm narrator", "")
		s.Refused(err, testutil.CodeVoiceNotCustom, "no sample for a voice set to "+voice)
	}
	builtin := Must(erin.VoiceSample("npc", "", ""))
	s.False(builtin.OwnVoice, "no voice style, no voice of its own")
	s.False(builtin.Stored, "a built-in voice's sample is not kept")
}

// An NPC game plays live or in rounds on a key that runs both: starting it
// asks which, and the player's choice starts it.
func (s *VoiceTestSuite) TestAnNPCGameAsksLiveOrRounds() {
	fay := s.CreateUser("voice-fay")
	Must(fay.AddApiKey("mock-voice-fay", "Fay's key", "mock"))
	game := Must(fay.UploadGame("bridge-troll"))

	_, err := fay.CreateSession(testutil.ForGame(game.Id.String()))
	s.Refused(err, testutil.CodeChooseMode)

	id := Must(fay.CreateSession(testutil.ForGame(game.Id.String()).InMode("rounds")))
	topology := Must(fay.SessionTopology(id))
	s.NotEmpty(topology.Nodes, "the session plays in rounds")
}
