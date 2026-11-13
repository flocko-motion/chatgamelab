package testutil

import (
	"net/url"
	"regexp"
	"strings"

	"cgl/api/routes"
	"cgl/mail"
)

// NewSignupClient returns a client that acts like a fresh Auth0 login of a person
// without an account (dev mode only).
func (s *BaseSuite) NewSignupClient() *UserClient {
	s.T().Helper()
	var resp routes.DevSignupTokenResponse
	if err := s.Public().Post("dev/signup-token", nil, &resp); err != nil {
		s.T().Fatalf("failed to get dev signup token: %v", err)
	}
	return &UserClient{Name: resp.Auth0ID, Token: resp.Token, t: s.T()}
}

// DevMails returns the mails the backend sent so far, newest first.
func (s *BaseSuite) DevMails() []mail.Message {
	s.T().Helper()
	var mails []mail.Message
	if err := s.Public().Get("dev/mails", &mails); err != nil {
		s.T().Fatalf("failed to read dev mails: %v", err)
	}
	return mails
}

var confirmLinkPattern = regexp.MustCompile(`/organization/confirm\?token=([^\s]+)`)

// LatestConfirmToken returns the token of the newest confirmation-link mail sent to the address.
func (s *BaseSuite) LatestConfirmToken(to string) string {
	s.T().Helper()
	for _, m := range s.DevMails() {
		if !strings.EqualFold(m.To, to) {
			continue
		}
		if match := confirmLinkPattern.FindStringSubmatch(m.Body); match != nil {
			token, err := url.QueryUnescape(match[1])
			if err != nil {
				s.T().Fatalf("bad token in link: %v", err)
			}
			return token
		}
	}
	s.T().Fatalf("no confirmation link mail to %s", to)
	return ""
}

// DevUserClient returns a client for a preseeded dev user (dev mode JWT by user ID).
func (s *BaseSuite) DevUserClient(userID, name string) *UserClient {
	s.T().Helper()
	var resp struct {
		Token string `json:"token"`
	}
	if err := s.Public().Get("users/"+userID+"/jwt", &resp); err != nil {
		s.T().Fatalf("failed to get JWT for dev user %s: %v", name, err)
	}
	return &UserClient{Name: name, ID: userID, Token: resp.Token, t: s.T()}
}
