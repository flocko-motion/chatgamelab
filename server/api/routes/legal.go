// package: routes / legal text HTTP handlers
// type:    logic
// job:     serves the current institution declaration and the youth-protection suggestions from the single embedded source (-> legal).
// limits:  read-only, public; no route registration (-> router.go).
package routes

import (
	"net/http"

	"cgl/api/httpx"
	"cgl/db"
	"cgl/legal"
	"cgl/obj"
)

// InstitutionDeclarationResponse is the current declaration plus the constraint suggestions
// shown next to it in the organisation registration form.
type InstitutionDeclarationResponse struct {
	Declaration                legal.Declaration                 `json:"declaration"`
	BindingLanguage            string                            `json:"bindingLanguage"`
	YouthProtectionSuggestions []legal.YouthProtectionSuggestion `json:"youthProtectionSuggestions"`
	// DefaultPromptConstraint is the site constraint for age 13+ (u13); the registration form starts with it.
	DefaultPromptConstraint *string `json:"defaultPromptConstraint,omitempty"`
	// MaxPromptConstraintLength is the maximum length of an organisation's restriction.
	MaxPromptConstraintLength int `json:"maxPromptConstraintLength"`
}

// GetInstitutionDeclaration godoc
//
//	@Summary		Get the institution declaration
//	@Description	Returns the current version of the Einverständniserklärung for organisations (binding German text, version, date, SHA-256 hash) and the suggested youth-protection prompt restrictions, the site default for age 13+ and the length limit. Public.
//	@Tags			legal
//	@Produce		json
//	@Success		200	{object}	InstitutionDeclarationResponse
//	@Router			/legal/institution-declaration [get]
func GetInstitutionDeclaration(w http.ResponseWriter, r *http.Request) {
	resp := InstitutionDeclarationResponse{
		Declaration:                legal.CurrentDeclaration(),
		BindingLanguage:            legal.BindingLanguage,
		YouthProtectionSuggestions: legal.YouthProtectionSuggestions(),
		MaxPromptConstraintLength:  obj.MaxPromptConstraintLength,
	}
	if settings, err := db.GetSystemSettings(r.Context()); err == nil {
		resp.DefaultPromptConstraint = settings.PromptConstraintU13
	}
	httpx.WriteJSON(w, http.StatusOK, resp)
}
