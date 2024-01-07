package api

import (
	"encoding/json"
	"webapp-server/db"
	"webapp-server/gpt"
	"webapp-server/obj"
	"webapp-server/router"
)

const (
	userAnonymous = uint(0)
)

type SessionNewRequest struct {
	GameId uint   `json:"gameId"`
	Hash   string `json:"hash"`
}

var SessionNew = router.NewEndpointJson(
	"/api/session/new",
	false,
	func(request router.Request) (interface{}, *obj.HTTPError) {
		var sessionNewRequest SessionNewRequest
		if err := json.NewDecoder(request.R.Body).Decode(&sessionNewRequest); err != nil {
			return nil, &obj.HTTPError{StatusCode: 400, Message: "Bad Request"}
		}

		// Note: public games are initialized via public hash, private games by game id
		var game *obj.Game
		var err *obj.HTTPError
		var userId uint
		if sessionNewRequest.GameId > 0 {
			if request.User == nil {
				return nil, &obj.HTTPError{StatusCode: 401, Message: "Unauthorized"}
			}
			if game, err = request.User.GetGame(sessionNewRequest.GameId); err != nil {
				return nil, err
			}
			userId = request.User.ID
		} else if sessionNewRequest.Hash != "" {
			if game, err = db.GetGameByPublicHash(sessionNewRequest.Hash); err != nil {
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

		// Create session
		if session, e = db.CreateSession(session); e != nil {
			return nil, &obj.HTTPError{StatusCode: 500, Message: "Internal Server Error"}
		}

		return session, nil
	},
)
