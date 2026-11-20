// package: testutil / the words for accounts and participants
// type:    logic
// job:     who is calling, profiles, roles, languages, statistics, and how participants join a workshop and sign in
// limits:  accounts only; institutions, workshops and invites live in orgs.go
package testutil

import (
	"fmt"
	"net/http"

	"cgl/apiclient"

	"github.com/google/uuid"
)

// WhoAmI reports who is calling: anonymous, registering, or registered with
// the account.
func (c caller) WhoAmI() (CurrentUser, error) {
	return decode[CurrentUser](c.api.GetApiUsersMeWithResponse(ctx()))
}

// GetMe is the caller's own account; a caller who isn't a registered
// account gets an error.
func (c caller) GetMe() (User, error) {
	me, err := c.WhoAmI()
	if err != nil {
		return User{}, err
	}
	if me.User == nil {
		return User{}, fmt.Errorf("the caller is %s, not a registered account", me.State)
	}
	return *me.User, nil
}

// GetRole is the user's role; the test fails if there is none.
func (u *UserClient) GetRole() string {
	u.t().Helper()
	me, err := u.GetMe()
	if err != nil {
		u.t().Fatalf("user %q: reading their account: %v", u.Name, err)
	}
	if me.Role == nil {
		u.t().Fatalf("user %q has no role", u.Name)
	}
	return me.Role.Role
}

// GetUser is another account's profile.
func (c caller) GetUser(userID string) (User, error) {
	return decode[User](c.api.GetApiUsersIdWithResponse(ctx(), userID))
}

// GetUsers lists every account.
func (c caller) GetUsers() ([]User, error) {
	return decode[[]User](c.api.GetApiUsersWithResponse(ctx()))
}

// UpdateUser changes an account's profile.
func (c caller) UpdateUser(userID string, update UserUpdate) (User, error) {
	return decode[User](c.api.PostApiUsersIdWithResponse(ctx(), userID, update))
}

// UpdateUserName renames an account; everything else stays.
func (c caller) UpdateUserName(userID, name string) (User, error) {
	return c.UpdateUser(userID, UserUpdate{Name: &name})
}

// SetUserAiQualityTier sets the user's own AI quality tier.
func (u *UserClient) SetUserAiQualityTier(tier string) error {
	_, err := u.UpdateUser(u.ID, UserUpdate{AiQualityTier: &tier})
	return err
}

// SetUserLanguage sets the caller's interface language.
func (c caller) SetUserLanguage(language string) error {
	return done(c.api.PatchApiUsersMeLanguageWithResponse(ctx(), apiclient.UpdateUserLanguageReq{Language: language}))
}

// SetUserRole gives an account a role, with no institution or workshop.
func (c caller) SetUserRole(userID, role string) (User, error) {
	return decode[User](c.api.PostApiUsersIdRoleWithResponse(ctx(), userID, apiclient.SetUserRoleReq{Role: role}))
}

// RemoveUserRole takes an account's role away.
func (c caller) RemoveUserRole(userID string) error {
	return done(c.api.DeleteApiUsersIdRoleWithResponse(ctx(), userID))
}

// DeleteUser deletes an account.
func (c caller) DeleteUser(userID string) error {
	return done(c.api.DeleteApiUsersIdWithResponse(ctx(), userID))
}

// GetStats is the caller's profile statistics.
func (c caller) GetStats() (UserStats, error) {
	return decode[UserStats](c.api.GetApiUsersMeStatsWithResponse(ctx()))
}

// GetParticipantCode is the caller's own participant code: empty for an
// account that isn't a participant.
func (c caller) GetParticipantCode() (ParticipantCode, error) {
	return decode[ParticipantCode](c.api.GetApiUsersMeParticipantCodeWithResponse(ctx()))
}

// --- Participants ---

// AcceptWorkshopInviteAnonymously accepts a workshop invite without signing
// in, which creates a participant and answers with their account and token.
func (s *BaseSuite) AcceptWorkshopInviteAnonymously(inviteToken string) (AcceptedInvite, error) {
	s.T().Helper()
	return s.Public().AcceptInviteByToken(inviteToken)
}

// JoinWorkshop accepts a workshop invite as a new participant and signs them
// in with the participant token they got, as they would sign in again.
func (s *BaseSuite) JoinWorkshop(inviteToken string) (*UserClient, error) {
	s.T().Helper()
	accepted, err := s.AcceptWorkshopInviteAnonymously(inviteToken)
	if err != nil {
		return nil, err
	}
	if accepted.AuthToken == nil {
		return nil, fmt.Errorf("accepting the invite answered no token: %s", accepted.Message)
	}
	return s.ParticipantLogin(*accepted.AuthToken)
}

// JoinWorkshopWithCookie accepts a workshop invite as a new participant, as
// a browser does, and answers the caller carrying only the session cookie
// the platform set.
func (s *BaseSuite) JoinWorkshopWithCookie(inviteToken string) (*CookieClient, error) {
	s.T().Helper()
	r, err := s.Public().api.PostApiInvitesIdOrTokenAcceptWithResponse(ctx(), inviteToken)
	if err := done(r, err); err != nil {
		return nil, err
	}
	cookies := r.HTTPResponse.Cookies()
	if len(cookies) == 0 {
		return nil, fmt.Errorf("accepting the invite set no cookie")
	}
	return &CookieClient{caller: newCaller(s, withCookie(cookies[0]), sameOrigin(s)), cookie: cookies[0]}, nil
}

// WithForgedCookie is a caller carrying a session cookie the platform never
// issued, named like a real one.
func (s *BaseSuite) WithForgedCookie(real *CookieClient) *CookieClient {
	s.T().Helper()
	forged := &http.Cookie{Name: real.cookie.Name, Value: "forged-" + uuid.NewString()}
	return &CookieClient{caller: newCaller(s, withCookie(forged), sameOrigin(s)), cookie: forged}
}

// signedInWithSession is the caller signed in with a session token, as
// participant-login answers it. A participant token is no session token: sign
// in with ParticipantLogin.
func (s *BaseSuite) signedInWithSession(token string) *UserClient {
	s.T().Helper()
	user := &UserClient{caller: newCaller(s, bearer(s, token)), Token: token}
	me, err := user.GetMe()
	if err != nil {
		s.T().Fatalf("reading the account behind a token: %v", err)
	}
	user.Name, user.ID = me.Name, me.Id.String()
	return user
}

// ParticipantLogin signs a participant in with their participant token, or
// its words, and answers them signed in with the session token they got.
func (s *BaseSuite) ParticipantLogin(participantToken string) (*UserClient, error) {
	s.T().Helper()
	resp, err := decode[apiclient.ParticipantLoginResp](s.Public().api.PostApiAuthParticipantLoginWithResponse(ctx(), apiclient.ParticipantLoginReq{Token: participantToken}))
	if err != nil {
		return nil, err
	}
	return s.signedInWithSession(resp.Token), nil
}

// AcceptWorkshopInviteAnonymouslyInLanguage accepts a workshop invite
// without signing in, as a browser whose Accept-Language header is
// acceptLanguage ("" sends none), which creates a participant.
func (s *BaseSuite) AcceptWorkshopInviteAnonymouslyInLanguage(inviteToken, acceptLanguage string) (AcceptedInvite, error) {
	s.T().Helper()
	var editors []apiclient.RequestEditorFn
	if acceptLanguage != "" {
		editors = append(editors, withAcceptLanguage(acceptLanguage))
	}
	return decode[AcceptedInvite](s.Public().api.PostApiInvitesIdOrTokenAcceptWithResponse(ctx(), inviteToken, editors...))
}

// withAcceptLanguage sends the languages a browser prefers.
func withAcceptLanguage(languages string) apiclient.RequestEditorFn {
	return header("Accept-Language", languages)
}
