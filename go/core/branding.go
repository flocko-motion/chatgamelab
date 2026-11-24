package core

import (
	"context"
	"encoding/json"
	"net/mail"
	"net/url"
	"strings"

	"cgl/core/internal/rules"
	"cgl/core/internal/store"
	"cgl/db"
	"cgl/log"
)

// The site's branding is how an instance presents itself: its name, its
// operator, its support address and the links the web app shows. The admin
// sets it; everyone reads it, also before signing in, since the login page
// shows it. A field the admin never set has its default, today's value; an
// empty link or address shows nothing.

// Branding is the site's branding as it applies.
type Branding struct {
	SiteName        string `json:"siteName"`        // the instance's name: page title, texts, mails
	OperatorName    string `json:"operatorName"`    // who runs the instance: footer, mails
	OperatorURL     string `json:"operatorUrl"`     // the operator's website
	SupportEmail    string `json:"supportEmail"`    // where users write for help
	WebsiteURL      string `json:"websiteUrl"`      // the instance's own website
	ContactURL      string `json:"contactUrl"`      // contact page
	PrivacyURL      string `json:"privacyUrl"`      // privacy policy
	TermsURL        string `json:"termsUrl"`        // terms of use
	EducatorInfoURL string `json:"educatorInfoUrl"` // using the instance for education
	HelpURL         string `json:"helpUrl"`         // help pages
	GameTipsURL     string `json:"gameTipsUrl"`     // tips for making games
	APIKeyFAQURL    string `json:"apiKeyFaqUrl"`    // questions about AI keys
	AIInsightsURL   string `json:"aiInsightsUrl"`   // what the AI insights show
	ReportGameURL   string `json:"reportGameUrl"`   // reporting a game
	CreditName      string `json:"creditName"`      // who programmed the software: footer
	CreditURL       string `json:"creditUrl"`       // their website
}

// DefaultBranding is what a field shows while the admin has not set it.
func DefaultBranding() Branding {
	return Branding{
		SiteName:        "ChatGameLab",
		OperatorName:    "JFF – Institut für Medienpädagogik",
		OperatorURL:     "https://jff.de",
		SupportEmail:    "chatgamelab@jff.de",
		WebsiteURL:      "https://chatgamelab.eu",
		ContactURL:      "https://chatgamelab.eu/kontakt/",
		PrivacyURL:      "https://chatgamelab.eu/datenschutz/",
		TermsURL:        "https://chatgamelab.eu/nutzungsbedingungen/",
		EducatorInfoURL: "https://chatgamelab.eu/edu/",
		HelpURL:         "https://chatgamelab.eu/hilfe/",
		GameTipsURL:     "https://chatgamelab.eu/hilfe/",
		APIKeyFAQURL:    "https://chatgamelab.eu/faq#api",
		AIInsightsURL:   "https://chatgamelab.eu/ai-insights",
		ReportGameURL:   "https://chatgamelab.eu/melden",
		CreditName:      "omnitopos.net",
		CreditURL:       "https://omnitopos.net",
	}
}

// BrandingChange is what changes in the site's branding, fields by their
// JSON name: Set gives fields a value (an empty one hides a link), Reset
// gives them back their default. Every other field stays.
type BrandingChange struct {
	Set   map[string]string // field (JSON name) → new value; empty hides a link
	Reset []string          // fields (JSON names) back to their default
}

// ReadBranding is the site's branding: {Read, SiteBranding, Everywhere},
// everyone.
func ReadBranding(ctx context.Context) (Branding, error) {
	if !granted(ctx, rules.Read, rules.SiteBranding, rules.Everywhere) {
		return Branding{}, forbidden("you may not read the site's branding")
	}
	stored, err := storedBranding(ctx, store.Queries())
	if err != nil {
		return Branding{}, err
	}
	return applyBranding(stored), nil
}

// EditBranding changes the site's branding, all or none: {Edit,
// SiteSettings, Everywhere}, the admin. It answers the branding as it now
// applies.
func EditBranding(ctx context.Context, c BrandingChange) (Branding, error) {
	if !granted(ctx, rules.Edit, rules.SiteSettings, rules.Everywhere) {
		return Branding{}, forbidden("only the admin changes the site's branding")
	}
	known := brandingFields(&Branding{})
	for field, value := range c.Set {
		if _, ok := known[field]; !ok {
			return Branding{}, refuse("unknown branding field %s", field)
		}
		if err := checkBrandingValue(field, strings.TrimSpace(value)); err != nil {
			return Branding{}, err
		}
	}
	for _, field := range c.Reset {
		if _, ok := known[field]; !ok {
			return Branding{}, refuse("unknown branding field %s", field)
		}
	}
	var applied Branding
	err := store.InTx(ctx, func(q *db.Queries) error {
		stored, err := storedBranding(ctx, q)
		if err != nil {
			return err
		}
		for field, value := range c.Set {
			stored[field] = strings.TrimSpace(value)
		}
		for _, field := range c.Reset {
			delete(stored, field)
		}
		raw, err := json.Marshal(stored)
		if err != nil {
			return err
		}
		if err := q.V2UpdateSiteBranding(ctx, raw); err != nil {
			return err
		}
		applied = applyBranding(stored)
		return nil
	})
	if err != nil {
		return Branding{}, err
	}
	log.Info("site branding changed", "by", account(ctx).UUID)
	return applied, nil
}

// storedBranding is what the admin set, by field.
func storedBranding(ctx context.Context, q *db.Queries) (map[string]string, error) {
	s, err := q.V2GetSystemSettings(ctx)
	if err != nil {
		return nil, err
	}
	stored := map[string]string{}
	if len(s.Branding) > 0 {
		if err := json.Unmarshal(s.Branding, &stored); err != nil {
			return nil, err
		}
	}
	return stored, nil
}

// applyBranding lays what the admin set over the defaults; a field that is
// no longer known is dropped.
func applyBranding(stored map[string]string) Branding {
	b := DefaultBranding()
	fields := brandingFields(&b)
	for field, value := range stored {
		if p, ok := fields[field]; ok {
			*p = value
		}
	}
	return b
}

// brandingFields are b's fields by their JSON name.
func brandingFields(b *Branding) map[string]*string {
	return map[string]*string{
		"siteName": &b.SiteName, "operatorName": &b.OperatorName, "operatorUrl": &b.OperatorURL,
		"supportEmail": &b.SupportEmail, "websiteUrl": &b.WebsiteURL, "contactUrl": &b.ContactURL,
		"privacyUrl": &b.PrivacyURL, "termsUrl": &b.TermsURL, "educatorInfoUrl": &b.EducatorInfoURL,
		"helpUrl": &b.HelpURL, "gameTipsUrl": &b.GameTipsURL, "apiKeyFaqUrl": &b.APIKeyFAQURL,
		"aiInsightsUrl": &b.AIInsightsURL, "reportGameUrl": &b.ReportGameURL,
		"creditName": &b.CreditName, "creditUrl": &b.CreditURL,
	}
}

// checkBrandingValue refuses a value its field cannot hold: the site needs a
// name, an address is an email address, a link a web address.
func checkBrandingValue(field, value string) error {
	switch {
	case field == "siteName":
		if value == "" {
			return refuse("the site needs a name")
		}
	case value == "":
		return nil
	case field == "supportEmail":
		if a, err := mail.ParseAddress(value); err != nil || a.Address != value {
			return refuse("the support address is no email address")
		}
	case strings.HasSuffix(field, "Url"):
		u, err := url.Parse(value)
		if err != nil || (u.Scheme != "http" && u.Scheme != "https") || u.Host == "" {
			return refuse("%s is no web address (http or https)", field)
		}
	}
	return nil
}
