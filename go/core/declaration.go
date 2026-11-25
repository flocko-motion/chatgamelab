package core

import (
	"context"

	"cgl/core/internal/store"
	"cgl/legal"
)

// DeclarationForm is what the form an organisation consents with shows: the
// current declaration, naming the site's operator, and the restriction it
// starts with.
type DeclarationForm struct {
	Declaration legal.Declaration
	// SiteRule is the site's restriction for 13–17 year olds without
	// parental consent, which an organisation's restriction starts from;
	// nil while the site has none.
	SiteRule *string
}

// ReadDeclarationForm is the declaration form, for anyone: the declaration
// is public, and so is the site's rule it starts from.
func ReadDeclarationForm(ctx context.Context) (DeclarationForm, error) {
	q := store.Queries()
	settings, err := q.V2GetSystemSettings(ctx)
	if err != nil {
		return DeclarationForm{}, err
	}
	stored, err := storedBranding(ctx, q)
	if err != nil {
		return DeclarationForm{}, err
	}
	return DeclarationForm{
		Declaration: legal.CurrentDeclaration().Of(applyBranding(stored).LegalOperator()),
		SiteRule:    siteRulesOf(settings).U13,
	}, nil
}
