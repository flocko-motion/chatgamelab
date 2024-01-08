package api

import (
	"context"
	"encoding/json"
	"github.com/sashabaranov/go-openai"
	"path"
	"webapp-server/db"
	"webapp-server/gpt"
	"webapp-server/obj"
	"webapp-server/router"
)

const (
	userAnonymous      = uint(0)
	sessionActionIntro = "intro"
	sessionActionInput = "input"
)

type SessionRequest struct {
	Action string `json:"action"` // type of action
	// creating a new session:
	GameId   uint   `json:"gameId"`
	GameHash string `json:"gameHash"`
	// playing a session:
	Message string `json:"message"` // user input
}

var SessionIntro = router.NewEndpointJson(
	"/api/session/",
	false,
	func(request router.Request) (interface{}, *obj.HTTPError) {
		sessionHash := path.Base(request.R.URL.Path)
		var body SessionRequest
		if err := json.NewDecoder(request.R.Body).Decode(&body); err != nil {
			return nil, &obj.HTTPError{StatusCode: 400, Message: "Bad Request"}
		}

		if sessionHash == "new" {
			return newSession(request, body)
		}

		var session *obj.Session
		var err error
		if session, err = db.GetSessionByHash(sessionHash); err != nil {
			return nil, &obj.HTTPError{StatusCode: 404, Message: "Not Found"}
		}

		ctx := context.Background()
		var message string
		switch body.Action {
		case sessionActionIntro:
			game, err := db.GetGameByID(session.GameID)
			if err != nil {
				return nil, &obj.HTTPError{StatusCode: 500, Message: "Internal Server Error"}
			}
			message, err = gpt.AddMessageToThread(ctx, *session, openai.ChatMessageRoleSystem, game.SessionStartSyscall)
		case sessionActionInput:
			message, err = gpt.AddMessageToThread(ctx, *session, openai.ChatMessageRoleUser, body.Message)
		}
		if err != nil {
			return nil, &obj.HTTPError{StatusCode: 500, Message: "Internal Server Error"}
		}

		return message, nil
	},
)

func newSession(request router.Request, body SessionRequest) (interface{}, *obj.HTTPError) {

	// Note: public games are initialized via public hash, private games by game id
	var game *obj.Game
	var err *obj.HTTPError
	var userId uint
	if body.GameId > 0 {
		if request.User == nil {
			return nil, &obj.HTTPError{StatusCode: 401, Message: "Unauthorized"}
		}
		if game, err = request.User.GetGame(body.GameId); err != nil {
			return nil, err
		}
		userId = request.User.ID
	} else if body.GameHash != "" {
		if game, err = db.GetGameByPublicHash(body.GameHash); err != nil {
			return nil, err
		}
		userId = userAnonymous
	} else {
		return nil, &obj.HTTPError{StatusCode: 400, Message: "Bad Request"}
	}

	// Build session
	session, e := gpt.CreateGameSession(game, userId)
	if e != nil {
		return nil, &obj.HTTPError{StatusCode: 500, Message: "Internal Server Error"}
	}

	// Store session
	if session, e = db.CreateSession(session); e != nil {
		return nil, &obj.HTTPError{StatusCode: 500, Message: "Internal Server Error"}
	}

	return session, nil
}
