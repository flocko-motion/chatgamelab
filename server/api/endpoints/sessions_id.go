package endpoints

import (
	"cgl/api/handler"
	"cgl/obj"

	"github.com/google/uuid"
)

const (
	userAnonymous = uint(0)
)

type SessionActionRequest struct {
	Message string `json:"message"` // user input
}

type SessionActionResponse struct {
	MessageID uuid.UUID `json:"messageId"`
}

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
	Session *obj.GameSession `json:"-"`
}

var Session = handler.NewEndpoint(
	"/api/sessions/{id:uuid}",
	handler.AuthOptional,
	"application/json",
	func(request handler.Request) (out interface{}, httpErr *obj.HTTPError) {
		return handleSessionRequest(request, false)
	},
)

func handleSessionRequest(request handler.Request, public bool) (out interface{}, httpErr *obj.HTTPError) {
	return nil, &obj.HTTPErrorNotImplemented
}
