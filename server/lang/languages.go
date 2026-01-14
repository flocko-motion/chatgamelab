package lang

var supportedLanguages = map[string]string{
	"fr": "French",
	"es": "Spanish",
	"it": "Italian",
	"pt": "Portuguese",
	"nl": "Dutch",
	"pl": "Polish",
	"ru": "Russian",
	"ja": "Japanese",
	"ko": "Korean",
	"zh": "Chinese",
	"ar": "Arabic",
	"hi": "Hindi",
	"tr": "Turkish",
	"sv": "Swedish",
	"da": "Danish",
	"no": "Norwegian",
	"fi": "Finnish",
}

// GetLanguageName returns the full name of a language from its code
func GetLanguageName(code string) string {
	if name, exists := supportedLanguages[code]; exists {
		return name
	}
	return code
}

// GetAllLanguageCodes returns a list of all supported language codes
func GetAllLanguageCodes() []string {
	codes := make([]string, 0, len(supportedLanguages))
	for code := range supportedLanguages {
		codes = append(codes, code)
	}
	return codes
}
