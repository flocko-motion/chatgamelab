package api

import (
	"encoding/json"
	"log"
	"path"
	"webapp-server/db"
	"webapp-server/gpt"
	"webapp-server/obj"
	"webapp-server/router"
)

const (
	userAnonymous = uint(0)
)

type SessionRequest struct {
	Action    string `json:"action"`    // type of action
	ChapterId uint   `json:"chapterId"` // id of action
	// creating a new session:
	GameID   uint   `json:"gameId"`
	GameHash string `json:"gameHash"`
	// playing a session:
	Message string            `json:"message"` // user input
	Status  []obj.StatusField `json:"status"`
	// context
	Game    *obj.Game    `json:"-"`
	Session *obj.Session `json:"-"`
}

var Session = router.NewEndpoint(
	"/api/session/",
	false,
	"application/json",
	func(request router.Request) (out interface{}, httpErr *obj.HTTPError) {
		sessionHash := path.Base(request.R.URL.Path)
		var sessionRequest SessionRequest
		if err := json.NewDecoder(request.R.Body).Decode(&sessionRequest); err != nil {
			return nil, &obj.HTTPError{StatusCode: 400, Message: "Bad Request"}
		}

		// TODO: we need to decide public/private and use the key of the game-owner for public games
		apiKey := request.User.OpenAiKeyPersonal

		if sessionHash == "new" {
			return newSession(request, sessionRequest, apiKey)
		}

		var err error
		if sessionRequest.Session, err = db.GetSessionByHash(sessionHash); err != nil {
			return nil, &obj.HTTPError{StatusCode: 404, Message: "Not Found"}
		}

		if sessionRequest.Game, err = db.GetGameByID(sessionRequest.Session.GameID); err != nil {
			return nil, &obj.HTTPError{StatusCode: 500, Message: "Internal Server Error"}
		}

		switch sessionRequest.Action {
		case obj.GameInputTypeIntro:
			return gpt.ExecuteAction(sessionRequest.Session, sessionRequest.Game, obj.GameActionInput{
				Type:      obj.GameInputTypeIntro,
				ChapterId: sessionRequest.ChapterId,
				Message:   sessionRequest.Game.SessionStartSyscall,
				Status:    sessionRequest.Game.StatusFields,
			}, apiKey)
		case obj.GameInputTypeAction:
			return gpt.ExecuteAction(sessionRequest.Session, sessionRequest.Game, obj.GameActionInput{
				Type:      obj.GameInputTypeAction,
				ChapterId: sessionRequest.ChapterId,
				Message:   sessionRequest.Message,
				Status:    sessionRequest.Status,
			}, apiKey)
		default:
			return nil, &obj.HTTPError{StatusCode: 400, Message: "Bad Request - unknown action: " + sessionRequest.Action}
		}
	},
)

func newSession(request router.Request, body SessionRequest, apiKey string) (*obj.Session, *obj.HTTPError) {

	// Note: public games are initialized via public hash, private games by game id
	var game *obj.Game
	var err *obj.HTTPError
	var userId uint
	if body.GameID > 0 {
		log.Printf("Creating new session for game id=%d", body.GameID)
		if request.User == nil {
			return nil, &obj.HTTPError{StatusCode: 401, Message: "Unauthorized"}
		}
		if game, err = request.User.GetGame(body.GameID); err != nil {
			return nil, err
		}
		userId = request.User.ID
	} else if body.GameHash != "" {
		log.Printf("Creating new session for game hash=%s", body.GameHash)
		if game, err = db.GetGameByPublicHash(body.GameHash); err != nil {
			return nil, err
		}
		userId = userAnonymous
	} else {
		log.Printf("Creating new session - no game id or hash provided")
		return nil, &obj.HTTPError{StatusCode: 400, Message: "Bad Request"}
	}

	// Build session
	session, e := gpt.CreateGameSession(game, userId, apiKey)
	if e != nil {
		return nil, &obj.HTTPError{StatusCode: 500, Message: "Internal Server Error"}
	}

	// Store session
	if session, e = db.CreateSession(session); e != nil {
		return nil, &obj.HTTPError{StatusCode: 500, Message: "Internal Server Error"}
	}

	return session, nil
}
