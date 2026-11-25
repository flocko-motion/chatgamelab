// package: legal / versioned legal texts
// type:    data
// job:     the organisations' declaration (Einverständniserklärung) and the youth-protection restrictions suggested beside it, embedded once, so form, mail and consent log read one source
// limits:  no HTTP (-> api), no persistence (-> core)
package legal

import (
	"crypto/sha256"
	"embed"
	"encoding/hex"
	"encoding/json"
	"fmt"
	"strings"
	"time"
)

//go:embed institution_declaration/*.md youth_protection_suggestions.json
var files embed.FS

// BindingLanguage is the language a declaration is agreed to in, whatever
// language the person reads the platform in.
const BindingLanguage = "de"

// MaxPromptConstraintLength is the longest an organisation's youth-protection
// restriction may be, in characters: kept short on purpose.
const MaxPromptConstraintLength = 200

// declarationVersions are the published versions, the current one last. To
// publish one, add institution_declaration/v<Version>.de.md and append it.
var declarationVersions = []struct {
	Version string
	Date    string // the legal text's date, YYYY-MM-DD
	// Consent is the binding sentence a person ticks to agree.
	Consent string
}{
	{
		Version: "1.0",
		Date:    "2026-11-12",
		Consent: "Ich habe die Einverständniserklärung gelesen und stimme ihr im Namen meiner Einrichtung zu. " +
			"Ich bestätige, dass ich berechtigt bin, diese Erklärung für meine Einrichtung abzugeben.",
	},
	{
		// names the instance's operator instead of one fixed provider
		Version: "1.1",
		Date:    "2026-11-25",
		Consent: "Ich habe die Einverständniserklärung gelesen und stimme ihr im Namen meiner Einrichtung zu. " +
			"Ich bestätige, dass ich berechtigt bin, diese Erklärung für meine Einrichtung abzugeben.",
	},
}

// operatorPlaceholder stands in a declaration for the operator's legal name,
// filled in by Of; the hash covers the text with the placeholder, so renaming
// the operator asks no organisation to consent again.
const operatorPlaceholder = "{{operatorLegalName}}"

// Declaration is one version of the declaration an organisation consents to,
// in the binding language.
type Declaration struct {
	Version string
	Date    string
	Text    string // Markdown; the first line is the title, in bold; the operator as a placeholder until Of
	Consent string // the sentence a person ticks to agree
	Hash    string // hex SHA-256 of Text with the placeholder, stored with every consent
}

// Of is d as the operator, by their legal name, offers it.
func (d Declaration) Of(operator string) Declaration {
	d.Text = strings.ReplaceAll(d.Text, operatorPlaceholder, operator)
	return d
}

// Title is the declaration's title, without its Markdown.
func (d Declaration) Title() string {
	title, _, _ := strings.Cut(d.Text, "\n")
	return strings.TrimSuffix(strings.TrimPrefix(title, "**"), "**")
}

// Body is the declaration after its title, as written.
func (d Declaration) Body() string {
	_, body, _ := strings.Cut(d.Text, "\n")
	return strings.TrimSpace(body)
}

// CurrentDeclaration is the version organisations consent to now.
func CurrentDeclaration() Declaration {
	d, _ := DeclarationByVersion(declarationVersions[len(declarationVersions)-1].Version)
	return d
}

// DeclarationByVersion is a published version; false for an unknown one.
func DeclarationByVersion(version string) (Declaration, bool) {
	for _, v := range declarationVersions {
		if v.Version != version {
			continue
		}
		raw, err := files.ReadFile(fmt.Sprintf("institution_declaration/v%s.%s.md", version, BindingLanguage))
		if err != nil {
			panic(fmt.Sprintf("legal: declaration v%s missing: %v", version, err))
		}
		text := strings.TrimSpace(string(raw))
		sum := sha256.Sum256([]byte(text))
		return Declaration{
			Version: version,
			Date:    v.Date,
			Text:    text,
			Consent: v.Consent,
			Hash:    hex.EncodeToString(sum[:]),
		}, true
	}
	return Declaration{}, false
}

// YouthProtectionSuggestion is a suggested youth-protection restriction
// (Jugendschutz-Prompt-Einschränkung) for players from an age on.
type YouthProtectionSuggestion struct {
	MinAge int    `json:"minAge"`
	Text   string `json:"text"`
}

// YouthProtectionSuggestions are the suggestions, youngest age first, as
// chatgamelab.eu publishes them.
func YouthProtectionSuggestions() []YouthProtectionSuggestion {
	raw, err := files.ReadFile("youth_protection_suggestions.json")
	if err != nil {
		panic("legal: youth_protection_suggestions.json missing: " + err.Error())
	}
	var parsed struct {
		Suggestions []YouthProtectionSuggestion `json:"suggestions"`
	}
	if err := json.Unmarshal(raw, &parsed); err != nil {
		panic("legal: youth_protection_suggestions.json invalid: " + err.Error())
	}
	return parsed.Suggestions
}

// ConsentStatus says whether an organisation's newest entry in the consent
// log covers the current declaration. It is pending while the organisation
// has none, has withdrawn it, or consented to an older version or text: the
// organisation is suspended until a head consents.
type ConsentStatus struct {
	CurrentVersion   string     `json:"currentVersion"`
	ConsentedVersion *string    `json:"consentedVersion,omitempty"` // the standing consent's
	ConsentedAt      *time.Time `json:"consentedAt,omitempty"`
	WithdrawnAt      *time.Time `json:"withdrawnAt,omitempty"` // when the newest consent was withdrawn
	Pending          bool       `json:"pending"`
}

// LogEntry is an entry of an organisation's consent log, as its status reads
// it: a consent, or the withdrawal of the consent before it.
type LogEntry struct {
	Withdrawal bool
	Version    string
	Hash       string
	At         time.Time
}

// ConsentStatusOf is the status of an organisation whose newest entry in the
// consent log is latest; nil: it has none.
func ConsentStatusOf(latest *LogEntry) ConsentStatus {
	current := CurrentDeclaration()
	status := ConsentStatus{CurrentVersion: current.Version, Pending: true}
	switch {
	case latest == nil:
	case latest.Withdrawal:
		status.WithdrawnAt = &latest.At
	default:
		status.ConsentedVersion = &latest.Version
		status.ConsentedAt = &latest.At
		status.Pending = latest.Version != current.Version || latest.Hash != current.Hash
	}
	return status
}
