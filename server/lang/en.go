package lang

var dict = map[string]string{
	"aiMessageStart":      "Start the game. Generate the opening scene.",
	"aiExpandPlotOutline": "Expand the plot outline into a full, detailed narrative. Write in second person perspective.",
}

// T returns the translation for the given key
func T(key string) string {
	if val, ok := dict[key]; ok {
		return val
	}
	return key // fallback to key itself
}
