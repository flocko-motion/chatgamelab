package engine

import "engine/internal/adapters"

// VoiceStore keeps the custom voices one game has on one key, so a voice is
// made once from the game's voice style and reused by every later session on
// that key. The platform implements it on its storage and hands it in with
// each offer; the engine asks it before it makes a voice, and waits while
// another session is making the same one.
type VoiceStore = adapters.VoiceStore

// MemoryVoices is a VoiceStore in memory, for a dev server and tests.
type MemoryVoices = adapters.MemoryVoices
