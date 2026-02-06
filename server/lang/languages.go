package lang

var supportedLanguages = map[string]string{
	"en":  "English",
	"de":  "Deutsch",
	"fr":  "Français",
	"es":  "Español",
	"it":  "Italiano",
	"pt":  "Português",
	"nl":  "Nederlands",
	"pl":  "Polski",
	"ru":  "Русский",
	"ja":  "日本語",
	"ko":  "한국어",
	"zh":  "中文",
	"ar":  "العربية",
	"hi":  "हिन्दी",
	"tr":  "Türkçe",
	"sv":  "Svenska",
	"da":  "Dansk",
	"no":  "Norsk",
	"fi":  "Suomi",
	"so":  "Soomaali",
	"ps":  "پښتو",
	"fa":  "فارسی",
	"uk":  "Українська",
	"bar": "Bayrisch",
	"el":  "Ελληνικά",
	"sr":  "Српски",
	"bs":  "Bosanski",
	"sq":  "Shqip",
	"bg":  "Български",
	"hu":  "Magyar",
	"hr":  "Hrvatski",
	"sl":  "Slovenščina",
	"cs":  "Čeština",
	"sk":  "Slovenčina",
	"ro":  "Română",
	"ti":  "ትግርኛ",
	"id":  "Bahasa Indonesia",
}

const TranslateInstruction = "You are an expert in translation of json structured language files for games. Translate the given JSON object to the target language while preserving the exact same structure and keys. Only translate the string values. Return a valid JSON object. You get the original already in two languages, so that you have more context to understand the intention of each field."

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

// IsValidLanguageCode checks if a language code is supported
func IsValidLanguageCode(code string) bool {
	_, exists := supportedLanguages[code]
	return exists
}
