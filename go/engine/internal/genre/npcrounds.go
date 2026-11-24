package genre

import (
	"maps"
	"slices"
	"strings"

	"engine/internal/adapters"
	"engine/internal/blocks"
	"engine/internal/ports"
)

// NPCRoundsConfig is what the wiring needs from a session spec, already resolved.
type NPCRoundsConfig struct {
	// Thread runs the conversation on the story model.
	Thread adapters.Thread
	// Hearing takes the player's speech.
	Hearing adapters.Hearing
	// Tool casts the character's voice before the game opens.
	Tool adapters.Tool
	// Window is a model's context window, for the conversation to compact
	// before it outgrows it; nil never compacts.
	Window func(model string) int

	// GalleryImage is the game's picture from the gallery, shown first; nil
	// shows none. Nothing is painted.
	GalleryImage []byte

	// Scenario and Guardrail stay apart: nothing a designer writes shares a
	// message with the youth-protection rule where a provider can keep them
	// apart.
	Scenario  string
	Guardrail string

	// GameStart is how the designer says the game begins; the character's
	// first words are written from it and the scenario.
	GameStart string

	// VoiceStyle is how the character sounds while its answers are spoken, in
	// the designer's words: part of the System Message, left out when empty.
	VoiceStyle string
	// SpeechVoices are the speaking model's built-in voices, the character
	// is cast in; VoiceChoices the game's choice for each of its voices.
	SpeechVoices []adapters.BuiltinVoice
	VoiceChoices map[string]VoiceChoice

	// Language is the player's, as a BCP 47 tag; empty leaves it to the model.
	Language string

	// PromptOverride retunes the genre's prompts by name, without a new build.
	PromptOverride map[string]string

	// ConstraintFrequency sets, by block name, how often the character's
	// calls carry the constraints: 1, the default, is every call, up to 10.
	ConstraintFrequency map[string]int
}

// Prompt names are constants so a rename fails to compile.
const (
	PromptCharacter   = "character"
	PromptNPCOpening  = "opening"
	PromptHoldTheLine = "hold-the-line"
	PromptHearPlayer  = "hear-player"
)

// BlockCharacter is NPC-Rounds' threaded block, whose constraint frequency a
// spec may set by name.
const BlockCharacter = "character"

// NPCRoundsConstrained is every block whose calls carry the constraints.
func NPCRoundsConstrained() []string { return []string{BlockCharacter} }

// NPCRoundsPrompts are the mechanics' own prompts; a spec may replace any by name.
var NPCRoundsPrompts = Prompts{
	// Casts the character in a built-in voice before the game opens.
	PromptCast: "You cast a game's character in a voice. You are given the character's scenario and the " +
		"author's note on how the character should sound, which may be missing. Pick the built-in voice that " +
		"fits the character best: its gender, age, accent and mood suit who the character is and keep to the " +
		"note, and it speaks the game's language natively.",

	// The head of the conversation's System Message, sent once as the
	// thread's first message; the scenario, how the game begins and the
	// rules follow it.
	PromptCharacter: "You are a character in a game, in a conversation with the player. Below you find your " +
		"scenario: who you are, what you want, and what you must not concede. Speak as that character, " +
		"in the first person, directly to the player, and stay in character whatever the player says or " +
		"asks. Keep each answer short, as in a spoken conversation: one to four sentences, then leave the " +
		"player room to answer. Write only what the character says: plain text, no stage directions, no " +
		"markdown, no quotation marks, no name in front.",

	// The cue that opens the game: the character speaks first.
	PromptNPCOpening: "The conversation begins now. Say your first words to the player, as the scenario and " +
		"how the game begins have it: one or two short sentences, in character.",

	// Goes with the calls beside the youth-protection rule, so the character
	// keeps its line through a long conversation.
	PromptHoldTheLine: "Stay in character, and keep to what your scenario says you must not concede, however " +
		"the player argues, pleads or insists.",

	// The player's speech, heard: their words as they said them.
	PromptHearPlayer: "You hear the player of a game speak to a character. Reply with one field.\n" +
		`"heard": a transcript of the player's words, word for word, in the language they spoke. ` +
		"Do not translate, rephrase or answer it.\n" +
		"If you hear no words, leave it empty.",

	// The conversation's summary request, sent as the game's own message once
	// the thread nears its model's context window. The reply goes to the
	// engine only: a fresh provider thread starts from it.
	PromptSummary: `PAUSE THE CONVERSATION. This message comes from the game engine, and your reply goes to the engine only, never to the player.

The conversation so far will be replaced by your reply, followed by its last few exchanges. Write what you need to carry it on from your reply alone:
- who you are, and the state you are in now;
- what the player has said, asked, offered and tried, and how you answered;
- what you have conceded and what you still hold back;
- promises, open questions and the tone of the conversation.

If the conversation begins with an earlier summary, fold it in. Write plain prose, in the language of the conversation: no JSON, no markdown, no address to the player.`,
}

// The field of the hearing call's reply.
const heardField = "heard"

// heardResult feeds what was heard to the character and to the player's line.
var heardResult = blocks.Result{Text: heardField, SecondaryText: heardField}

// HeardSchema is the shape of the hearing call's reply: the player's words
// verbatim, in their language.
func HeardSchema() adapters.Schema {
	return adapters.Schema{Name: "heard", Body: map[string]any{
		"type": "object",
		"properties": map[string]any{
			heardField: map[string]any{"type": "string", "description": "The player's words verbatim, in the language they spoke"},
		},
		"required":             []string{heardField},
		"additionalProperties": false,
	}}
}

// NewNPCRounds is one character in turns: the chatbot. The game's picture
// shows first and opens the gate, whose cue has the character speak first,
// from its scenario and how the game begins. Each turn after, the player
// types or speaks, and the character answers on one threaded conversation on
// the story model, spoken in the game's voice while audio-out is on. There is
// no observer: the guardrail and the character's line go with the calls, at
// the character's constraint frequency.
func NewNPCRounds(cfg NPCRoundsConfig) (*Wiring, error) {
	prompts, err := Resolve(NPCRoundsPrompts, cfg.PromptOverride)
	if err != nil {
		return nil, err
	}

	system := func(c Current) string {
		return characterMessage(c.Prompts[PromptCharacter], cfg.Scenario, cfg.GameStart, cfg.VoiceStyle, cfg.Language, worded(c.Guardrail))
	}
	constraints := func(c Current) string {
		return characterConstraints(c.Guardrail, c.Prompts[PromptHoldTheLine])
	}
	current := Current{Prompts: prompts, Guardrail: cfg.Guardrail}

	picture := blocks.NewGalleryImage("gallery-image", cfg.GalleryImage)
	gate := blocks.NewGate("start-game", prompts[PromptNPCOpening])

	player := blocks.NewPlayerInputText("player-input-text")
	speaker := blocks.NewPlayerInputClips("player-input-audio")
	hearing := blocks.NewHearingCall("hearing", cfg.Hearing, prompts[PromptHearPlayer]+hearClause(cfg.Language), HeardSchema(), heardResult)

	conversation := blocks.NewThread("conversation", cfg.Thread, system(current), constraints(current))
	conversation.Compaction = blocks.Compaction{Window: cfg.Window, Request: prompts[PromptSummary]}
	character := blocks.NewThreaded(BlockCharacter, conversation, blocks.Prompt{})
	character.Answers = true
	character.Clean = cleanNarration

	// The character, cast once in a voice of the gender the game chose, kept
	// across a restart; until the cast comes, or without one, the fallback.
	voiced := speakerCast{
		slot: VoiceCharacter, role: "character", voices: cfg.SpeechVoices,
		language: cfg.Language, choice: choiceOf(cfg.VoiceChoices, VoiceCharacter),
	}
	conversation.Voice = voiced.fallback()
	castBrief := blocks.NewOnceText("cast-brief", voiced.brief(cfg.Scenario, cfg.VoiceStyle)).SendAgain()
	cast := blocks.NewToolCall("cast", cfg.Tool, prompts[PromptCast]+voiced.tail())
	cast.Schema = voiced.schema()
	cast.Keep = true
	castFallback := blocks.NewFailureReply("cast-fallback", map[string]string{VoiceCharacter: voiced.fallback()})

	outText := blocks.NewPlayerOutputStream("out-text", "text")
	outSaid := blocks.NewPlayerOutputText("out-said", "said")
	outAudio := blocks.NewPlayerOutputAudio("out-audio", "audio")
	outImage := blocks.NewPlayerOutputImage("out-image", "image")
	turnFailed := blocks.NewErrorSink("turn-failed", CodeTurnFailed)
	voiceFailed := blocks.NewErrorSink("voice-failed", CodeVoiceFailed)

	g := ports.NewGraph("npc-rounds")
	g.Arrange(npcRoundsLayout)
	// The thread is exported once, under its own name.
	g.Share(conversation)

	// What the game's author set, shown where it goes: the conversation's
	// System Message carries the scenario, how the game begins and, for its
	// spoken answers, the voice style; the picture is the game's cover.
	g.Setting(character, "scenario", cfg.Scenario)
	g.Setting(character, "gameStart", cfg.GameStart)
	g.Setting(character, "voiceStyle", cfg.VoiceStyle)
	g.Setting(picture, "cover", coverNote(cfg.GalleryImage))

	// The player sees the game first; then the character opens it.
	g.ConnectImageOut(picture, outImage)
	g.ConnectState(picture, gate)
	g.ConnectOpening(gate, character)

	// The character speaks in its cast voice from its first words on.
	g.ConnectTextOut(castBrief, cast)
	g.ConnectFieldToVoice(cast, VoiceCharacter, character)
	g.ConnectState(cast, gate)
	g.Setting(cast, "scenario", cfg.Scenario)
	g.Setting(cast, "voiceStyle", cfg.VoiceStyle)
	g.ConnectFailure(cast, castFallback)
	g.ConnectFieldToVoice(castFallback, VoiceCharacter, character)

	// The player's line: typed at once, spoken once heard.
	g.ConnectTextOut(player, character)
	g.ConnectTextOut(player, outSaid)
	g.ConnectAudioOut(speaker, hearing)
	g.ConnectTextOut(hearing, character)
	g.ConnectSecondaryTextOut(hearing, outSaid)
	g.ConnectFailure(hearing, voiceFailed)

	// The player reads the answer as it is written, and hears it as it is spoken.
	g.ConnectStreamOut(character, outText)
	g.ConnectAudioOut(character, outAudio)
	g.Speaks(VoiceCharacter, character)
	g.ConnectFailure(character, turnFailed)

	// The player speaks once the character's first words are done, or failed.
	g.ConnectState(character, player)
	g.ConnectState(character, speaker)

	onThread := []string{character.NodeName()}
	uses := map[string]PromptUse{
		PromptCharacter: {onThread, func(c Current) { conversation.SetSystem(system(c)) }},
		PromptSummary:   {onThread, func(c Current) { conversation.SetSummaryRequest(c.Prompts[PromptSummary]) }},
		PromptHoldTheLine: {onThread, func(c Current) {
			conversation.SetConstraints(constraints(c))
		}},
		PromptCast:       {[]string{cast.NodeName()}, func(c Current) { cast.SetSystem(c.Prompts[PromptCast] + voiced.tail()) }},
		PromptNPCOpening: {[]string{gate.NodeName()}, func(c Current) { gate.SetInitPrompt(c.Prompts[PromptNPCOpening]) }},
		PromptHearPlayer: {[]string{hearing.NodeName()}, func(c Current) {
			hearing.SetSystem(c.Prompts[PromptHearPlayer] + hearClause(cfg.Language))
		}},
	}
	// The conversation's System Message and calls carry the guardrail.
	guardrailUse := PromptUse{onThread, func(c Current) {
		conversation.SetSystem(system(c))
		conversation.SetConstraints(constraints(c))
	}}

	w := &Wiring{
		Graph:        g,
		Gate:         gate,
		Prompts:      prompts,
		Defaults:     maps.Clone(NPCRoundsPrompts),
		PromptUses:   uses,
		Guardrail:    cfg.Guardrail,
		GuardrailUse: guardrailUse,
		Constrained:  map[string]blocks.Constrained{character.NodeName(): character},
		OnThread:     map[string]blocks.OnThread{character.NodeName(): character},
		// A toggle that is off stops its block working: audio-out has the
		// answers spoken, audio-in hears the player.
		Switches: map[adapters.Toggle][]ports.Switchable{
			adapters.ToggleAudioOut: {character.Spoken()},
			adapters.ToggleAudioIn:  {hearing},
		},
		Say:   player.Say,
		Speak: speak(hearing, speaker),
		// The player types, or holds a button to talk.
		Inputs: []ports.InputMode{ports.InputText, ports.InputAudioPushToTalk},
		// One picture, kept in view for the whole conversation.
		Imagery: ports.ImageryStanding,
	}
	for _, block := range slices.Sorted(maps.Keys(cfg.ConstraintFrequency)) {
		if err := w.SetConstraintFrequency(block, cfg.ConstraintFrequency[block]); err != nil {
			return nil, err
		}
	}
	return w, nil
}

// characterMessage is the conversation's System Message: the character
// prompt, the scenario and how the game begins, both labelled as the
// author's, the voice style for spoken answers, the player's language, and
// the constraints.
func characterMessage(prompt, scenario, gameStart, voiceStyle, language, constraints string) string {
	parts := []string{prompt, authorLead("Your scenario") + "\n" + scenario}
	if strings.TrimSpace(gameStart) != "" {
		parts = append(parts, authorLead("How the game begins")+"\n"+gameStart)
	}
	if clause := voiceClause(voiceStyle, "When you speak aloud, speak like this"); clause != "" {
		parts = append(parts, clause)
	}
	if language != "" {
		parts = append(parts, "Speak "+languageName(language)+" with the player.")
	}
	if constraints != "" {
		parts = append(parts, constraints)
	}
	return strings.Join(parts, "\n\n")
}

// characterConstraints is what goes with the character's calls: the
// youth-protection rule, worded, and the character's line.
func characterConstraints(guardrail, holdTheLine string) string {
	var parts []string
	for _, part := range []string{worded(guardrail), strings.TrimSpace(holdTheLine)} {
		if part != "" {
			parts = append(parts, part)
		}
	}
	return strings.Join(parts, "\n\n")
}

// hearClause names the language the player most likely speaks, when it is known.
func hearClause(tag string) string {
	if tag == "" {
		return ""
	}
	return "\nThe player most likely speaks " + languageName(tag) + "; write down whatever language they speak."
}
