package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// ReviewTestSuite tests the editor's review: one point about a game's form,
// paid by the caller's own key, saved game or not.
type ReviewTestSuite struct {
	testutil.BaseSuite
}

func TestReviewSuite(t *testing.T) {
	s := &ReviewTestSuite{}
	s.SuiteName = "Review Tests"
	suite.Run(t, s)
}

func reviewForm() testutil.ReviewForm {
	return testutil.ReviewForm{
		Genre: "adventure", Name: "Lighthouse", Description: "Keep the light burning through the storm.",
		SystemMessageScenario:  "The player keeps a lighthouse; the goal is to guide three ships home.",
		SystemMessageGameStart: "A storm rises.", ImageStyle: "watercolour",
	}
}

// An empty required field is the point, named by the game's own field name.
func (s *ReviewTestSuite) TestAMissingFieldIsThePoint() {
	frida := s.CreateUser("review-frida")
	form := reviewForm()
	form.SystemMessageGameStart = ""
	review := Must(frida.ReviewGame(form))
	s.Equal("systemMessageGameStart", review.Field)
	s.True(review.Missing)
}

// A full form asks the model, which on the mock finds nothing to fix.
func (s *ReviewTestSuite) TestAFullFormIsReviewedOnTheCallersKey() {
	gus := s.CreateUser("review-gus")
	Must(gus.AddApiKey("mock-review-gus", "Gus's key", "mock"))
	review := Must(gus.ReviewGame(reviewForm()))
	s.Equal("", review.Field)
	s.False(review.Missing)
	s.NotEmpty(review.Text)
}

// Without a key there is no review; the editor says so quietly.
func (s *ReviewTestSuite) TestWithoutAKeyThereIsNoReview() {
	hana := s.CreateUser("review-hana")
	_, err := hana.ReviewGame(reviewForm())
	s.Refused(err, testutil.CodeNoApiKey)
}
