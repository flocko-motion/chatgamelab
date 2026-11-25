package testing

import (
	"testing"
	"time"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// SystemSettingsTestSuite tests admin system settings management
type SystemSettingsTestSuite struct {
	testutil.BaseSuite
}

func TestSystemSettingsSuite(t *testing.T) {
	s := &SystemSettingsTestSuite{}
	s.SuiteName = "System Settings Tests"
	suite.Run(t, s)
}

// TestAdminsCanSetAndReplaceFreeUseApiKey tests that multiple admins can
// set, replace, and clear the system free-use API key, and that non-admins are denied.
func (s *SystemSettingsTestSuite) TestAdminsCanSetAndReplaceFreeUseApiKey() {
	// Use preseeded admin as admin1
	admin1 := s.DevUser()

	// Create a second admin
	admin2 := s.CreateUser("admin2")
	s.Role(admin2, "admin")
	s.Equal("admin", admin2.GetRole())
	s.T().Logf("Created admin2 with admin role")

	// Both admins add a mock API key
	admin1Key := Must(admin1.AddApiKey("mock-key-admin1", "Admin1 Mock Key", "openai"))
	s.NotEmpty(admin1Key.Id)
	s.NotEmpty(admin1Key.ApiKeyId)
	s.T().Logf("Admin1 added API key: shareID=%s, apiKeyID=%s", admin1Key.Id, admin1Key.ApiKeyId)

	admin2Key := Must(admin2.AddApiKey("mock-key-admin2", "Admin2 Mock Key", "openai"))
	s.NotEmpty(admin2Key.Id)
	s.NotEmpty(admin2Key.ApiKeyId)
	s.T().Logf("Admin2 added API key: shareID=%s, apiKeyID=%s", admin2Key.Id, admin2Key.ApiKeyId)

	// Verify initial state: no free-use key set
	settings := Must(admin1.GetSystemSettings())
	s.Nil(settings.FreeUseApiKeyId, "initially no free-use key should be set")
	s.T().Logf("Initial settings: no free-use key")

	// Admin1 sets his key as the system free-use key
	admin1KeyIDStr := admin1Key.ApiKeyId.String()
	settings = Must(admin1.SetSystemFreeUseApiKey(&admin1KeyIDStr))
	s.NotNil(settings.FreeUseApiKeyId, "free-use key should be set")
	s.Equal(admin1Key.ApiKeyId, *settings.FreeUseApiKeyId)
	s.Require().NotNil(settings.FreeUseApiKeyName)
	s.Equal("Admin1 Mock Key", *settings.FreeUseApiKeyName)
	s.Require().NotNil(settings.FreeUseApiKeyPlatform)
	s.Equal("openai", *settings.FreeUseApiKeyPlatform)
	s.T().Logf("Admin1 set free-use key to his own key")

	// Admin2 can see the free-use key is set (admins see full details)
	settings = Must(admin2.GetSystemSettings())
	s.NotNil(settings.FreeUseApiKeyId, "admin2 should see free-use key")
	s.Equal(admin1Key.ApiKeyId, *settings.FreeUseApiKeyId)
	s.T().Logf("Admin2 can see admin1's free-use key in settings")

	// Admin2 replaces it with his own key
	admin2KeyIDStr := admin2Key.ApiKeyId.String()
	settings = Must(admin2.SetSystemFreeUseApiKey(&admin2KeyIDStr))
	s.NotNil(settings.FreeUseApiKeyId, "free-use key should be set to admin2's key")
	s.Equal(admin2Key.ApiKeyId, *settings.FreeUseApiKeyId)
	s.Require().NotNil(settings.FreeUseApiKeyName)
	s.Equal("Admin2 Mock Key", *settings.FreeUseApiKeyName)
	s.Require().NotNil(settings.FreeUseApiKeyPlatform)
	s.Equal("openai", *settings.FreeUseApiKeyPlatform)
	s.T().Logf("Admin2 replaced free-use key with his own key")

	// Admin1 sees admin2's key is now the free-use key
	settings = Must(admin1.GetSystemSettings())
	s.NotNil(settings.FreeUseApiKeyId)
	s.Equal(admin2Key.ApiKeyId, *settings.FreeUseApiKeyId)
	s.Require().NotNil(settings.FreeUseApiKeyName)
	s.Equal("Admin2 Mock Key", *settings.FreeUseApiKeyName)
	s.T().Logf("Admin1 sees admin2's key is now the free-use key")

	// Admin2 clears the free-use key
	settings = Must(admin2.SetSystemFreeUseApiKey(nil))
	s.Nil(settings.FreeUseApiKeyId, "free-use key should be cleared")
	s.Empty(settings.FreeUseApiKeyName)
	s.Empty(settings.FreeUseApiKeyPlatform)
	s.T().Logf("Admin2 cleared the free-use key")

	// Non-admin cannot set free-use key
	regularUser := s.CreateUser("regular-user")
	s.Equal("individual", regularUser.GetRole())
	_, err := regularUser.SetSystemFreeUseApiKey(&admin1KeyIDStr)
	s.Forbidden(err)
	s.T().Logf("Non-admin correctly denied setting free-use key")
}

// TestDeleteApiKeyWhileSetAsFreeUse tests that an admin can always delete
// their own API key even when it is currently set as the system free-use key.
// The system settings should be automatically cleaned up (free-use key cleared).
func (s *SystemSettingsTestSuite) TestDeleteApiKeyWhileSetAsFreeUse() {
	admin1 := s.DevUser()

	// Create a second admin
	admin2 := s.CreateUser("admin2-delete-test")
	s.Role(admin2, "admin")

	// Admin1 adds a key and sets it as free-use
	admin1Key := Must(admin1.AddApiKey("mock-key-delete-test", "Key To Delete", "openai"))
	admin1KeyIDStr := admin1Key.ApiKeyId.String()
	settings := Must(admin1.SetSystemFreeUseApiKey(&admin1KeyIDStr))
	s.NotNil(settings.FreeUseApiKeyId)
	s.Equal(admin1Key.ApiKeyId, *settings.FreeUseApiKeyId)
	s.T().Logf("Admin1 set free-use key: %s", admin1Key.ApiKeyId)

	// Admin1 deletes the key (cascade) — should succeed even though it's the free-use key
	MustSucceed(admin1.DeleteApiKey(admin1Key.Id.String(), true))
	s.T().Logf("Admin1 deleted API key that was set as free-use key")

	// Verify system settings were cleaned up: free-use key should be nil
	settings = Must(admin1.GetSystemSettings())
	s.Nil(settings.FreeUseApiKeyId, "free-use key should be cleared after key deletion")
	s.Empty(settings.FreeUseApiKeyName)
	s.Empty(settings.FreeUseApiKeyPlatform)
	s.T().Logf("System settings correctly cleared after key deletion")

	// Admin2 also sees the free-use key is cleared
	settings = Must(admin2.GetSystemSettings())
	s.Nil(settings.FreeUseApiKeyId, "admin2 should also see free-use key cleared")
	s.T().Logf("Admin2 confirms free-use key is cleared")

	// Now test the same with admin2's key
	admin2Key := Must(admin2.AddApiKey("mock-key-admin2-delete", "Admin2 Key To Delete", "openai"))
	admin2KeyIDStr := admin2Key.ApiKeyId.String()
	settings = Must(admin2.SetSystemFreeUseApiKey(&admin2KeyIDStr))
	s.NotNil(settings.FreeUseApiKeyId)
	s.Equal(admin2Key.ApiKeyId, *settings.FreeUseApiKeyId)
	s.T().Logf("Admin2 set free-use key: %s", admin2Key.ApiKeyId)

	// Admin2 deletes the key — should succeed and clean up settings
	MustSucceed(admin2.DeleteApiKey(admin2Key.Id.String(), true))
	s.T().Logf("Admin2 deleted API key that was set as free-use key")

	// Verify cleanup
	settings = Must(admin1.GetSystemSettings())
	s.Nil(settings.FreeUseApiKeyId, "free-use key should be cleared after admin2's key deletion")
	s.T().Logf("System settings correctly cleared after admin2's key deletion")
}

// TestHeadCannotModifySystemSettings verifies that a user with the "head" role
// can neither read nor modify system settings: they are for admins only.
func (s *SystemSettingsTestSuite) TestHeadCannotModifySystemSettings() {
	admin := s.DevUser()

	// Create an institution and invite a user as head
	inst := Must(admin.CreateInstitution("test-inst-settings"))
	head := s.CreateUser("head-settings")
	invite := Must(admin.InviteToInstitution(inst.Id.String(), "head", head.ID))
	Must(head.AcceptInvite(invite.Id.String()))
	s.Equal("head", head.GetRole())
	s.T().Logf("Created head user in institution")

	// Head cannot read system settings
	_, err := head.GetSystemSettings()
	s.Forbidden(err, "head should not be able to read system settings")
	s.T().Logf("Head correctly denied reading system settings")

	// Head cannot set free-use key
	adminKey := Must(admin.AddApiKey("mock-key-head-test", "Admin Key", "openai"))
	adminKeyIDStr := adminKey.ApiKeyId.String()
	_, err = head.SetSystemFreeUseApiKey(&adminKeyIDStr)
	s.Forbidden(err, "head should not be able to set free-use key")
	s.T().Logf("Head correctly denied setting free-use key")

	// Head cannot clear free-use key
	_, err = head.SetSystemFreeUseApiKey(nil)
	s.Forbidden(err, "head should not be able to clear free-use key")
	s.T().Logf("Head correctly denied clearing free-use key")
}

// TestFreeUseKeyClearedWhenAdminRoleRemoved tests that when an admin who set
// the system free-use API key loses their admin role, the free-use key is
// automatically cleared from system settings.
func (s *SystemSettingsTestSuite) TestFreeUseKeyClearedWhenAdminRoleRemoved() {
	admin1 := s.DevUser()

	// Create a second admin who will have their role removed
	admin2 := s.CreateUser("admin2-role-removal")
	s.Role(admin2, "admin")
	s.Equal("admin", admin2.GetRole())
	s.T().Logf("Created admin2 with admin role")

	// Admin2 adds a key and sets it as the system free-use key
	admin2Key := Must(admin2.AddApiKey("mock-key-role-test", "Admin2 Role Test Key", "openai"))
	admin2KeyIDStr := admin2Key.ApiKeyId.String()
	settings := Must(admin2.SetSystemFreeUseApiKey(&admin2KeyIDStr))
	s.NotNil(settings.FreeUseApiKeyId)
	s.Equal(admin2Key.ApiKeyId, *settings.FreeUseApiKeyId)
	s.T().Logf("Admin2 set free-use key: %s", admin2Key.ApiKeyId)

	// Admin1 removes admin2's admin role — user becomes individual
	MustSucceed(admin1.RemoveUserRole(admin2.ID))
	s.Equal("individual", admin2.GetRole())
	s.T().Logf("Admin1 removed admin2's admin role")

	// The free-use key should be cleared because admin2 is no longer an admin
	settings = Must(admin1.GetSystemSettings())
	s.Nil(settings.FreeUseApiKeyId, "free-use key should be cleared when the key owner loses admin role")
	s.Empty(settings.FreeUseApiKeyName)
	s.Empty(settings.FreeUseApiKeyPlatform)
	s.T().Logf("System free-use key correctly cleared after admin role removal")
}

// The free-use key is one of the admin's own: someone else's key does not
// exist for them, and the site's key stays as it was.
func (s *SystemSettingsTestSuite) TestAdminSetsOnlyTheirOwnKeyAsFreeUse() {
	admin := s.DevUser()
	other := s.CreateUser("key-owner-free-use")
	key := Must(other.AddApiKey("mock-key-not-admins", "Someone's key", "openai")).ApiKeyId.String()
	before := Must(admin.GetSystemSettings())

	_, err := admin.SetSystemFreeUseApiKey(&key)
	s.NotFound(err, "an admin may not make someone else's key pay for the site")
	s.Equal(before.FreeUseApiKeyId, Must(admin.GetSystemSettings()).FreeUseApiKeyId, "the site's key stays")
}

// A site rule is the site's default until the admin changes it, and the
// default again once they clear it or set it to the default's text: the site
// rule is never empty.
func (s *SystemSettingsTestSuite) TestSiteRulesStartAsTheDefaultAndGoBack() {
	admin := s.DevUser()
	settings := Must(admin.GetSystemSettings())
	def := settings.PromptConstraintDefaults.U18
	s.NotEmpty(def)
	s.Require().NotNil(settings.PromptConstraintU18)
	s.Equal(def, *settings.PromptConstraintU18, "an unset level is its default")

	own := "Nur Rätsel, keine Kämpfe."
	settings = Must(admin.UpdateSystemSettings(testutil.SettingsUpdate{PromptConstraintU18: &own}))
	s.Equal(own, *settings.PromptConstraintU18)

	cleared := ""
	settings = Must(admin.UpdateSystemSettings(testutil.SettingsUpdate{PromptConstraintU18: &cleared}))
	s.Require().NotNil(settings.PromptConstraintU18)
	s.Equal(def, *settings.PromptConstraintU18, "a cleared level is its default again")

	settings = Must(admin.UpdateSystemSettings(testutil.SettingsUpdate{PromptConstraintU18: &def}))
	s.Equal(def, *settings.PromptConstraintU18)
	s.Equal(0, s.DatabaseCount("SELECT count(*) FROM system_settings WHERE prompt_constraint_u18 IS NOT NULL"),
		"the default's text stores nothing, so the default keeps applying")
}

// --- A fresh instance and its mail ---

// operators are names a fresh instance must not show: the software is
// neutral, and an instance's operator brands it.
var operators = []string{"JFF", "Jugend Film Fernsehen", "Aktion Jugendschutz", "chatgamelab.eu"}

// A fresh instance names no operator: its branding is the site's name and
// the software's credit, and the declaration names no one.
func (s *SystemSettingsTestSuite) TestAFreshInstanceNamesNoOperator() {
	site := Must(s.Public().GetSite())
	s.Equal("ChatGameLab", site.Branding.SiteName)
	s.Empty(site.Branding.OperatorName)
	s.Empty(site.Branding.OperatorLegalName)
	s.Empty(site.Branding.SupportEmail)
	s.Empty(site.Branding.PrivacyUrl)

	branding := testutil.AsSent(site.Branding)
	declaration := Must(s.Public().GetDeclaration())
	for _, name := range operators {
		s.NotContains(branding, name, "the branding")
		s.NotContains(declaration.Declaration.Text, name, "the declaration")
	}
}

// The declaration names the site's operator by their legal name, else by
// their name.
func (s *SystemSettingsTestSuite) TestTheDeclarationNamesTheOperator() {
	admin := s.DevUser()
	Must(admin.SetSiteBranding(map[string]string{"operatorName": "Muster", "operatorLegalName": ""}))
	s.Contains(Must(s.Public().GetDeclaration()).Declaration.Text, "Datenschutzerklärung von Muster (Anbieter)")

	Must(admin.SetSiteBranding(map[string]string{"operatorLegalName": "Verein Muster e.V."}))
	s.Contains(Must(s.Public().GetDeclaration()).Declaration.Text, "Datenschutzerklärung von Verein Muster e.V. (Anbieter)")
}

// The admin sets the mail server; its password is written, never read back.
func (s *SystemSettingsTestSuite) TestTheMailServerKeepsItsPassword() {
	admin := s.DevUser()
	mail := Must(admin.GetSystemSettings()).Mail
	s.Empty(mail.Host, "a fresh instance only logs mails")
	s.Equal(587, mail.Port)
	s.False(mail.PasswordSet)

	password := "geheim-123"
	set := func(m testutil.MailSettingsReq) testutil.MailSettings {
		return Must(admin.UpdateSystemSettings(testutil.SettingsUpdate{Mail: &m})).Mail
	}
	mail = set(testutil.MailSettingsReq{Host: "smtp.example.org", Port: 465, User: "post@example.org",
		From: "Muster <noreply@example.org>", ConsentCopyEmail: "kopie@example.org", Password: &password})
	s.Equal("smtp.example.org", mail.Host)
	s.Equal(465, mail.Port)
	s.Equal("Muster <noreply@example.org>", mail.From)
	s.Equal("kopie@example.org", mail.ConsentCopyEmail)
	s.True(mail.PasswordSet)

	s.NotContains(testutil.AsSent(Must(admin.GetSystemSettings())), password, "no answer carries the password")

	mail = set(testutil.MailSettingsReq{Host: "smtp.example.org", Port: 587, User: "post@example.org"})
	s.True(mail.PasswordSet, "a change without a password keeps it")
	s.Empty(mail.ConsentCopyEmail)

	cleared := ""
	mail = set(testutil.MailSettingsReq{Password: &cleared})
	s.False(mail.PasswordSet, "an empty password clears it")
	s.Empty(mail.Host, "and an empty host only logs mails again")

	for name, bad := range map[string]testutil.MailSettingsReq{
		"a port out of range": {Host: "smtp.example.org", Port: 70000},
		"a host with a port":  {Host: "smtp.example.org:25"},
		"a broken sender":     {Host: "smtp.example.org", From: "kein Absender"},
		"a broken copy":       {Host: "smtp.example.org", ConsentCopyEmail: "kopie"},
	} {
		_, err := admin.UpdateSystemSettings(testutil.SettingsUpdate{Mail: &bad})
		s.Invalid(err, name)
	}

	head := s.CreateUser("head")
	_, err := head.UpdateSystemSettings(testutil.SettingsUpdate{Mail: &testutil.MailSettingsReq{Host: "evil.example.org"}})
	s.Forbidden(err, "only the admin sets the mail server")
}

// --- Session retention ---

// retention sets the site's session retention, in hours.
func (s *SystemSettingsTestSuite) retention(hours int) {
	s.T().Helper()
	settings := Must(s.DevUser().UpdateSystemSettings(testutil.SettingsUpdate{SessionRetentionHours: &hours}))
	s.Require().Equal(hours, settings.SessionRetentionHours)
}

// session is a session of a fresh player.
func (s *SystemSettingsTestSuite) session() string {
	s.T().Helper()
	player := s.CreateUser(testutil.Unique("idle"))
	Must(player.AddApiKey("mock-"+testutil.Unique("k"), testutil.Unique("Key"), "mock"))
	game := Must(player.UploadGame("alien-first-contact"))
	return Must(player.CreateSession(testutil.ForGame(game.Id.String())))
}

// idleFor makes the sessions look not played for those many hours. The
// backend restarts first: the engine stores a session it plays as it goes,
// and after a restart it holds none.
func (s *SystemSettingsTestSuite) idleFor(hours map[string]int) {
	s.T().Helper()
	s.RestartBackend()
	for id, h := range hours {
		s.SessionIdleFor(id, h)
	}
}

// janitorPass restarts the backend, whose janitor makes its first pass at
// start, and waits until done answers true.
func (s *SystemSettingsTestSuite) janitorPass(done func() bool) {
	s.T().Helper()
	s.RestartBackend()
	deadline := time.Now().Add(15 * time.Second)
	for !done() && time.Now().Before(deadline) {
		time.Sleep(200 * time.Millisecond)
	}
	s.Require().True(done(), "the janitor's pass is done")
}

// A site keeps sessions not played for 48 hours until the admin sets another
// window.
func (s *SystemSettingsTestSuite) TestSessionRetentionIs48HoursByDefault() {
	s.Equal(48, Must(s.DevUser().GetSystemSettings()).SessionRetentionHours)
}

// The retention is whole hours, 0 or more.
func (s *SystemSettingsTestSuite) TestSessionRetentionIsNeverNegative() {
	hours := -1
	_, err := s.DevUser().UpdateSystemSettings(testutil.SettingsUpdate{SessionRetentionHours: &hours})
	s.Refused(err, testutil.CodeValidation)
	s.Equal(48, Must(s.DevUser().GetSystemSettings()).SessionRetentionHours)
}

// The janitor deletes the sessions not played for the window the admin set.
func (s *SystemSettingsTestSuite) TestTheJanitorDeletesSessionsByTheStoredWindow() {
	defer s.retention(48)
	s.retention(2)
	stale, recent := s.session(), s.session()
	s.idleFor(map[string]int{stale: 3, recent: 1})

	s.janitorPass(func() bool { return !s.SessionStored(stale) })

	s.True(s.SessionStored(recent), "a session played within the window stays")
}

// A window of 0 keeps every session.
func (s *SystemSettingsTestSuite) TestRetentionZeroKeepsEverySession() {
	defer s.retention(48)
	s.retention(0)
	old := s.session()
	s.idleFor(map[string]int{old: 1000})
	// the same pass removes sign-ups left for a day, after the sessions
	signUp := s.CreateRegisteringUser()
	s.SignUpStartedHoursAgo(signUp.ID, 25)

	s.janitorPass(func() bool { return !s.AccountStored(signUp.ID) })

	s.True(s.SessionStored(old), "with retention 0 the janitor deletes no session")
}
