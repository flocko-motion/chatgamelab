// package: api / schemaf endpoints
// type:    logic
// job:     serves the declaration organisations consent to, with the youth-protection restrictions suggested beside it, from the one embedded source
// limits:  read-only and public; the texts live in legal; the site's own restriction is read from v1's system settings
package api

import (
	"context"

	"cgl/db"
	"cgl/legal"
	"cgl/log"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// GetDeclarationEndpoint returns the declaration organisations consent to.
//
// Public. The current version of the Einverständniserklärung: its binding
// German text (Markdown, title in bold on the first line), the sentence a
// person ticks to agree, version, date and SHA-256 hash; the suggested
// youth-protection restrictions; the site's restriction for 13–17 year olds
// without parental consent, which an organisation's form starts with; and
// the longest restriction an organisation may set.
type GetDeclarationEndpoint struct{}

func (GetDeclarationEndpoint) Method() string             { return "GET" }
func (GetDeclarationEndpoint) Path() string               { return "/api/legal/institution-declaration" }
func (GetDeclarationEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthNone }
func (GetDeclarationEndpoint) Handle(ctx context.Context, _ GetDeclarationReq) (DeclarationResp, error) {
	d := legal.CurrentDeclaration()
	resp := DeclarationResp{
		Declaration: DeclarationView{
			Version: d.Version,
			Date:    d.Date,
			Text:    d.Text,
			Consent: d.Consent,
			Hash:    d.Hash,
		},
		BindingLanguage:            legal.BindingLanguage,
		YouthProtectionSuggestions: legal.YouthProtectionSuggestions(),
		MaxPromptConstraintLength:  legal.MaxPromptConstraintLength,
	}
	settings, err := db.GetSystemSettings(ctx)
	if err != nil {
		log.Warn("declaration: site restriction unavailable", "error", err)
		return resp, nil
	}
	resp.DefaultPromptConstraint = settings.PromptConstraintU13
	return resp, nil
}

// GetDeclarationReq takes no parameters.
type GetDeclarationReq struct{}

// DeclarationResp is the current declaration and what the organisation's
// form shows beside it.
type DeclarationResp struct {
	Declaration                DeclarationView                   `json:"declaration"`
	BindingLanguage            string                            `json:"bindingLanguage"`
	YouthProtectionSuggestions []legal.YouthProtectionSuggestion `json:"youthProtectionSuggestions"`
	// DefaultPromptConstraint is the site's restriction for 13–17 year olds
	// without parental consent; absent while the site has none.
	DefaultPromptConstraint   *string `json:"defaultPromptConstraint,omitempty"`
	MaxPromptConstraintLength int     `json:"maxPromptConstraintLength"`
}

// DeclarationView is one version of the declaration, in its binding
// language.
type DeclarationView struct {
	Version string `json:"version"`
	Date    string `json:"date"`
	Text    string `json:"text"`
	Consent string `json:"consent"`
	Hash    string `json:"hash"`
}
