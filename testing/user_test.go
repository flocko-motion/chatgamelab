package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

type UserTestSuite struct {
	testutil.BaseSuite
}

func (s *UserTestSuite) SetupSuite() {
	s.BaseSuite.SetupSuite()
}

func TestUserTestSuite(t *testing.T) {
	suite.Run(t, new(UserTestSuite))
}

// TestUserLanguagePreference tests setting and retrieving user language preference
func (s *UserTestSuite) TestUserLanguagePreference() {
	// Create test users for this test
	clientAlice := s.CreateUser("alice-lang-test")
	clientBob := s.CreateUser("bob-lang-test")

	// Get current user info - should have default language (empty or "en")
	user := Must(clientAlice.GetMe())
	s.T().Logf("Initial user language: '%s'", user.Language)

	// Set language to German
	err := clientAlice.SetUserLanguage("de")
	s.NoError(err, "Setting language should succeed")
	s.T().Logf("Set user language to: de")

	// Verify language was updated
	user = Must(clientAlice.GetMe())
	s.Equal("de", user.Language, "User language should be updated to 'de'")
	s.T().Logf("Verified user language: %s", user.Language)

	// Set language to French
	err = clientAlice.SetUserLanguage("fr")
	s.NoError(err, "Setting language should succeed")
	s.T().Logf("Set user language to: fr")

	// Verify language was updated again
	user = Must(clientAlice.GetMe())
	s.Equal("fr", user.Language, "User language should be updated to 'fr'")
	s.T().Logf("Verified user language: %s", user.Language)

	// Verify Bob's language is independent
	userBob := Must(clientBob.GetMe())
	s.NotEqual("fr", userBob.Language, "Bob's language should be independent from Alice's")
	s.T().Logf("Bob's language (independent): '%s'", userBob.Language)
}
