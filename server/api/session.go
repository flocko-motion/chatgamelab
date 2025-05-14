package api

import (
	"encoding/json"
	"fmt"
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
		return handleSessionRequest(request, false)
	},
)

func handleSessionRequest(request router.Request, public bool) (out interface{}, httpErr *obj.HTTPError) {
	var err error
	var apiKey string

	sessionHash := path.Base(request.R.URL.Path)
	var sessionRequest SessionRequest
	if err = json.NewDecoder(request.R.Body).Decode(&sessionRequest); err != nil {
		return nil, &obj.HTTPError{StatusCode: 400, Message: "Bad Request"}
	}

	if sessionHash == "new" {
		if apiKey, httpErr = getGamePublicApiKey(sessionRequest.GameID, request.User, public); httpErr != nil {
			return nil, httpErr
		}
		return newSession(request, sessionRequest.GameID, apiKey)
	}

	if sessionRequest.Session, err = db.GetSessionByHash(sessionHash); err != nil {
		return nil, &obj.HTTPError{StatusCode: 404, Message: "Not Found"}
	}

	if sessionRequest.Game, err = db.GetGameByID(sessionRequest.Session.GameID); err != nil {
		return nil, &obj.HTTPError{StatusCode: 500, Message: "Internal Server Error"}
	}

	if apiKey, httpErr = getGamePublicApiKey(sessionRequest.Game.ID, request.User, public); httpErr != nil {
		return nil, httpErr
	}

	var (
		response *obj.GameActionOutput
	)

	switch sessionRequest.Action {
	case obj.GameInputTypeIntro:
		response, httpErr = gpt.ExecuteAction(sessionRequest.Session, sessionRequest.Game, obj.GameActionInput{
			Type:      obj.GameInputTypeIntro,
			ChapterId: sessionRequest.ChapterId,
			Message:   sessionRequest.Game.SessionStartSyscall,
			Status:    sessionRequest.Game.StatusFields,
		}, apiKey)
	case obj.GameInputTypeAction:
		response, httpErr = gpt.ExecuteAction(sessionRequest.Session, sessionRequest.Game, obj.GameActionInput{
			Type:      obj.GameInputTypeAction,
			ChapterId: sessionRequest.ChapterId,
			Message:   sessionRequest.Message,
			Status:    sessionRequest.Status,
		}, apiKey)
	default:
		response, httpErr = nil, &obj.HTTPError{StatusCode: 400, Message: "Bad Request - unknown action: " + sessionRequest.Action}
	}

	report := obj.SessionUsageReport{
		SessionID: sessionRequest.Session.ID,
		ApiKey:    apiKey[:8] + "..",
		GameID:    sessionRequest.Game.ID,
		UserID:    sessionRequest.Session.UserID,
		Action:    sessionRequest.Action,
		Error:     fmt.Sprintf("%v", err),
	}
	if request.User != nil {
		report.UserName = request.User.Name
	} else {
		report.UserName = "[public]"
	}
	db.WriteSessionUsageReport(report)

	return response, httpErr
}

func getGamePublicApiKey(gameID uint, user *db.User, public bool) (string, *obj.HTTPError) {
	var apiKey string
	if public {
		game, err := db.GetGameByID(gameID)
		if err != nil {
			return "", &obj.HTTPError{StatusCode: 500, Message: "Not found - failed to get game"}
		}

		if !game.SharePlayActive {
			return "", &obj.HTTPError{StatusCode: 404, Message: "Not Found"}
		}

		var owner *db.User
		owner, err = db.GetUserByID(game.UserId)
		if err != nil {
			return "", &obj.HTTPError{StatusCode: 500, Message: "Internal Server Error - failed to get owner of public game"}
		}
		log.Printf("Owner of public game: %+v", owner)
		apiKey = owner.OpenAiKeyPublish
		return owner.OpenAiKeyPublish, nil
	} else {
		apiKey = user.OpenAiKeyPersonal
	}
	if apiKey == "" {
		return "", &obj.HTTPError{StatusCode: 401, Message: "Unauthorized - missing API key for session"}
	}
	return apiKey, nil
}

func newSession(request router.Request, gameID uint, apiKey string) (*obj.Session, *obj.HTTPError) {
	var game *obj.Game
	var userId uint
	if gameID > 0 {
		log.Printf("Creating new session for game id=%d", gameID)
		var err error
		if game, err = db.GetGameByID(gameID); err != nil {
			return nil, &obj.HTTPError{StatusCode: 404, Message: "Not Found - Game not found"}
		}
		if request.User == nil {
			userId = userAnonymous
		} else {
			userId = request.User.ID
		}
	} else {
		log.Printf("Creating new session - no game id or hash provided")
		return nil, &obj.HTTPError{StatusCode: 400, Message: "Bad Request"}
	}

	// Build session
	session, e := gpt.CreateGameSession(game, userId, apiKey)
	if e != nil {
		return nil, &obj.HTTPError{StatusCode: 500, Message: e.Error()}
	}

	// Store session
	if session, e = db.CreateSession(session); e != nil {
		return nil, &obj.HTTPError{StatusCode: 500, Message: "Internal Server Error"}
	}

	return session, nil
}
