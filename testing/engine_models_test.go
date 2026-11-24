package testing

import (
	"testing"

	"cgl/testing/testutil"

	"github.com/stretchr/testify/suite"
)

// The admin assigns the engine's models and keeps its model table (#589).
type EngineModelsTestSuite struct {
	testutil.BaseSuite
}

func TestEngineModelsSuite(t *testing.T) {
	s := &EngineModelsTestSuite{}
	s.SuiteName = "Engine models"
	suite.Run(t, s)
}

// A model enters a tier only from the model table: assigning one the table
// lacks is refused; once added, it runs the task; while it does, it cannot
// leave the table; resetting the task brings back the default, and then the
// model can go.
func (s *EngineModelsTestSuite) TestAModelEntersATierOnlyFromTheModelTable() {
	admin := s.DevUser()
	before := Must(admin.EngineModels())
	codeModel, assigned := testutil.AssignedModel(before, "openai", "medium", "story")
	s.NotEmpty(codeModel)
	s.False(assigned)

	_, err := admin.AssignEngineModel("openai", "medium", "story", "gpt-next")
	s.Invalid(err, "a model the table lacks")

	models := Must(admin.AddTextModel("openai", "gpt-next", 1, 4, 400_000))
	row, listed := testutil.ModelRow(models, "openai", "gpt-next")
	s.True(listed)
	s.False(row.InUse)
	s.Nil(row.Default, "the code has no entry for it")

	models = Must(admin.AssignEngineModel("openai", "medium", "story", "gpt-next"))
	model, assigned := testutil.AssignedModel(models, "openai", "medium", "story")
	s.Equal("gpt-next", model)
	s.True(assigned)
	row, _ = testutil.ModelRow(models, "openai", "gpt-next")
	s.True(row.InUse)

	_, err = admin.ResetEngineModelEntry("openai", "gpt-next")
	s.Invalid(err, "a model in use leaves the table")

	models = Must(admin.AssignEngineModel("openai", "medium", "story", ""))
	model, assigned = testutil.AssignedModel(models, "openai", "medium", "story")
	s.Equal(codeModel, model)
	s.False(assigned)

	models = Must(admin.ResetEngineModelEntry("openai", "gpt-next"))
	_, listed = testutil.ModelRow(models, "openai", "gpt-next")
	s.False(listed)
}

// A model must fit its task: a text model cannot hear the player.
func (s *EngineModelsTestSuite) TestAModelMustFitItsTask() {
	admin := s.DevUser()
	Must(admin.AddTextModel("openai", "gpt-deaf", 1, 1, 100_000))
	_, err := admin.AssignEngineModel("openai", "low", "hearing", "gpt-deaf")
	s.Invalid(err)
	Must(admin.ResetEngineModelEntry("openai", "gpt-deaf"))
}

// A code model's entry changed by the admin is marked, and its reset brings
// the code's figures back.
func (s *EngineModelsTestSuite) TestTheAdminRepricesACodeModel() {
	admin := s.DevUser()
	models := Must(admin.AddTextModel("openai", "gpt-5.6-luna", 9, 9, 1_050_000))
	row, _ := testutil.ModelRow(models, "openai", "gpt-5.6-luna")
	s.True(row.Overridden)
	s.Equal("admin", row.Facts.Source)
	s.NotNil(row.Default)

	models = Must(admin.ResetEngineModelEntry("openai", "gpt-5.6-luna"))
	row, _ = testutil.ModelRow(models, "openai", "gpt-5.6-luna")
	s.False(row.Overridden)
	s.NotEqual("admin", row.Facts.Source)
}

// Mistral's models are shown, not changed.
func (s *EngineModelsTestSuite) TestOnlyConfigurablePlatformsChange() {
	admin := s.DevUser()
	_, err := admin.AssignEngineModel("mistral", "low", "story", "mistral-small-latest")
	s.Invalid(err)
}

// A tier's pictures take the admin's quality; resetting the tiers brings the
// default model and quality back.
func (s *EngineModelsTestSuite) TestTheAdminSetsAPictureQualityAndResetsTheTiers() {
	admin := s.DevUser()
	before := Must(admin.EngineModels())
	model, _ := testutil.AssignedModel(before, "openai", "high", "image")
	_, def := testutil.PictureQuality(before, "openai", "high")

	_, err := admin.AssignPicture("openai", "high", model, "ultra")
	s.Invalid(err, "an unknown quality")

	models := Must(admin.AssignPicture("openai", "high", model, "medium"))
	quality, _ := testutil.PictureQuality(models, "openai", "high")
	s.Equal("medium", quality)

	models = Must(admin.ResetEngineModels("openai", "assignments"))
	quality, _ = testutil.PictureQuality(models, "openai", "high")
	s.Equal(def, quality)
	_, assigned := testutil.AssignedModel(models, "openai", "high", "image")
	s.False(assigned)
}

// Resetting the model table gives code models their figures back and removes
// added models no tier runs.
func (s *EngineModelsTestSuite) TestResettingTheModelTable() {
	admin := s.DevUser()
	Must(admin.AddTextModel("openai", "gpt-5.6-luna", 9, 9, 1_050_000))
	Must(admin.AddTextModel("openai", "gpt-spare", 1, 1, 100_000))

	models := Must(admin.ResetEngineModels("openai", "entries"))
	row, _ := testutil.ModelRow(models, "openai", "gpt-5.6-luna")
	s.False(row.Overridden)
	_, listed := testutil.ModelRow(models, "openai", "gpt-spare")
	s.False(listed)
}
