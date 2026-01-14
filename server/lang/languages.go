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
