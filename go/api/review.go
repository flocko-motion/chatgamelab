// package: api / schemaf endpoints
// type:    logic
// job:     has a game's form reviewed for the editor: one point to improve
// limits:  reviewing is session/review.go's and the engine's; nothing is stored
package api

import (
	"context"
	"strconv"
	"unicode/utf8"

	"cgl/session"
	"engine"

	schemafapi "github.com/flocko-motion/schemaf/api"
)

// maxReviewText caps each field of a form sent for a review, in characters.
const maxReviewText = 20000

// ReviewGameEndpoint reviews a game's form as the editor holds it, saved or
// not, and answers the one point most worth improving: the field it is
// about (title, description, scenario, gameStart, imageStyle, voiceStyle,
// statusFields), empty when there is nothing to fix; missing when a
// required field is empty, which the client says in its own words; else one
// short sentence in the caller's language. Nothing is stored, and nothing
// blocks saving.
//
// The caller pays with their own keys, as in the editor: workshop,
// organisation, own, site; never a game's sponsor. None: 400 no_api_key; a
// key its provider rejects: 400 invalid_api_key; a provider that fails or
// takes too long: 503 ai_error.
type ReviewGameEndpoint struct{}

func (ReviewGameEndpoint) Method() string             { return "POST" }
func (ReviewGameEndpoint) Path() string               { return "/api/games/review" }
func (ReviewGameEndpoint) Auth() schemafapi.AuthLevel { return schemafapi.AuthRequired }
func (ReviewGameEndpoint) Handle(ctx context.Context, req ReviewGameReq) (ReviewGameResp, error) {
	for _, field := range []string{req.Name, req.Description, req.Scenario, req.GameStart, req.ImageStyle, req.VoiceStyle, req.StatusFields} {
		if utf8.RuneCountInString(field) > maxReviewText {
			return ReviewGameResp{}, errBadRequest("a field is longer than " + strconv.Itoa(maxReviewText) + " characters")
		}
	}
	review, err := session.ReviewGameForm(ctx, engine.ReviewSpec{
		Genre: engine.Genre(req.Genre), Title: req.Name, Description: req.Description, Scenario: req.Scenario,
		GameStart: req.GameStart, ImageStyle: req.ImageStyle, VoiceStyle: req.VoiceStyle, StatusFields: req.StatusFields,
	})
	if err != nil {
		return ReviewGameResp{}, err
	}
	return ReviewGameResp{Field: gameFieldNames[review.Field], Missing: review.Missing, Text: review.Text}, nil
}

// gameFieldNames are the engine's names for a form's fields, as a game has
// them; a name it lacks, such as none, is empty.
var gameFieldNames = map[string]string{
	"title": "name", "description": "description", "scenario": "systemMessageScenario",
	"gameStart": "systemMessageGameStart", "imageStyle": "imageStyle", "voiceStyle": "voiceStyle",
	"statusFields": "statusFields",
}

// ReviewGameReq is the form as the editor holds it, by the names a game has.
type ReviewGameReq struct {
	Genre        string `json:"genre"`
	Name         string `json:"name"`
	Description  string `json:"description"`
	Scenario     string `json:"systemMessageScenario"`
	GameStart    string `json:"systemMessageGameStart"`
	ImageStyle   string `json:"imageStyle"`
	VoiceStyle   string `json:"voiceStyle"`
	StatusFields string `json:"statusFields"`
}

// ReviewGameResp is the one point: the field, by the name a game has it
// under (name, description, systemMessageScenario, ...), empty for nothing
// to fix; missing for an empty required field; the sentence otherwise.
type ReviewGameResp struct {
	Field   string `json:"field"`
	Missing bool   `json:"missing"`
	Text    string `json:"text"`
}
