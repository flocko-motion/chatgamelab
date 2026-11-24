package openai

import "engine/internal/adapters"

// LiveVoices are GPT-Live's built-in voices as the provider lists them in its
// guide ("Voice options", Managing GPT-Live sessions), as of 24 November
// 2026: each with the language it speaks natively, its accent and its
// presentation. The provider offers no list to query, so this one is kept by
// hand. A gender left empty is one the guide does not state.
var LiveVoices = []adapters.BuiltinVoice{
	// The provider's default; its presentation is how listeners hear it, not stated.
	{Name: "marin", Language: "en", Accent: "North American", Gender: "feminine"},
	{Name: "gleam", Language: "en", Accent: "North American", Gender: "feminine"},
	{Name: "meridian", Language: "en", Accent: "North American", Gender: "masculine"},
	{Name: "delta", Language: "en", Accent: "Southern U.S.", Gender: "feminine"},
	{Name: "cinder", Language: "en", Accent: "Southern U.S.", Gender: "masculine"},
	{Name: "vesper", Language: "en", Accent: "British", Gender: "masculine"},
	{Name: "halo", Language: "en", Accent: "British", Gender: "feminine"},
	{Name: "willow", Language: "en", Accent: "Irish", Gender: "feminine"},
	{Name: "stone", Language: "en", Accent: "Irish", Gender: "masculine"},
	{Name: "quartz", Language: "en", Accent: "Australian", Gender: "feminine"},
	{Name: "pearl", Language: "en", Accent: "Australian", Gender: "feminine"},
	{Name: "ripple", Language: "en", Accent: "Australian", Gender: "masculine"},
	{Name: "beacon", Language: "en", Accent: "Filipino", Gender: "masculine"},
	{Name: "nuri", Language: "de", Accent: "German", Gender: "feminine"},
	{Name: "juni", Language: "de", Accent: "German", Gender: "feminine"},
	{Name: "flitz", Language: "de", Accent: "German", Gender: "masculine"},
	{Name: "brise", Language: "fr", Accent: "French", Gender: "feminine"},
	{Name: "sillage", Language: "fr", Accent: "French", Gender: "masculine"},
	{Name: "calandria", Language: "es", Accent: "Argentine"},
	{Name: "compas", Language: "es", Accent: "Argentine"},
	{Name: "hornero", Language: "es", Accent: "Argentine"},
	{Name: "luciernaga", Language: "es", Accent: "Mexican"},
	{Name: "destello", Language: "es", Accent: "Mexican"},
	{Name: "bossa", Language: "pt", Accent: "Brazilian", Gender: "feminine"},
	{Name: "tempo", Language: "pt", Accent: "Brazilian", Gender: "masculine"},
	{Name: "nira", Language: "hi", Accent: "Indian", Gender: "feminine"},
	{Name: "harema", Language: "ja", Accent: "Japanese", Gender: "feminine"},
	{Name: "shida", Language: "ja", Accent: "Japanese", Gender: "masculine"},
	{Name: "noeul", Language: "ko", Accent: "Korean", Gender: "feminine"},
}
