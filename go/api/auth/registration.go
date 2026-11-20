// package: auth / who is calling
// type:    logic
// job:     the details that complete an account's registration, and their rules
// limits:  no persistence (-> db.CompleteRegistration); no HTTP (-> api)
package auth

import (
	"fmt"
	"net/mail"
	"strings"
	"unicode/utf8"

	"cgl/constants"
	"cgl/obj"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// maxNameLength is the longest name, in characters.
const maxNameLength = 24

// Registration is what completes an account: a name and the age group from the
// Altersabfrage. The email is optional.
type Registration struct {
	Name  string  `json:"name"`
	Email *string `json:"email,omitempty"`
	// AgeGroup is "u13" (13–17 without parental consent), "u13p" (13–17 with
	// parental consent) or "u18" (18 and over).
	AgeGroup string `json:"ageGroup"`
	// Language is the ISO 639-1 code of the interface the person registered
	// in; the account keeps its language when it is empty.
	Language string `json:"language,omitempty"`
}

// Validate checks the details and returns them trimmed, with a blank email
// dropped. Every refusal wraps ErrBadRequest.
func (r Registration) Validate() (Registration, error) {
	r.Name = strings.TrimSpace(r.Name)
	switch {
	case r.Name == "":
		return r, badRequest("a name is required")
	case utf8.RuneCountInString(r.Name) > maxNameLength:
		return r, badRequest(fmt.Sprintf("a name has at most %d characters", maxNameLength))
	case constants.IsProfane(r.Name):
		return r, badRequest("this name is not allowed")
	}

	if r.Email != nil {
		email := strings.TrimSpace(*r.Email)
		if email == "" {
			r.Email = nil
		} else if _, err := mail.ParseAddress(email); err != nil {
			return r, badRequest("invalid email address")
		} else {
			r.Email = &email
		}
	}

	switch r.AgeGroup {
	case obj.AgeGroupU13, obj.AgeGroupU13p, obj.AgeGroupU18:
	case "":
		return r, badRequest("an age group is required")
	default:
		return r, badRequest("the age group is u13, u13p or u18")
	}

	r.Language = strings.TrimSpace(r.Language)
	return r, nil
}

func badRequest(msg string) error {
	return fmt.Errorf("%s: %w", msg, schemafapi.ErrBadRequest)
}
