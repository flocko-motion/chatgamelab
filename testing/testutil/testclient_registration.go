package testutil

import (
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
