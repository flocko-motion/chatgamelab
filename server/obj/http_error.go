package obj

import (
	"encoding/json"
	"fmt"
)

var (
	HTTPErrorNotImplemented = HTTPError{StatusCode: 501, Message: "Not Implemented"}
)

type HTTPError struct {
	StatusCode int
	Code       string // Machine-readable error code for frontend
	Message    string
}

func (e HTTPError) Error() string {
	return e.Message
}

func NewHTTPErrorWithCode(statusCode int, code string, message string) *HTTPError {
	return &HTTPError{StatusCode: statusCode, Code: code, Message: message}
}

func (e HTTPError) Json() []byte {
	type Error struct {
		Type  string `json:"type"`
		Error string `json:"error"`
	}
	resObj := Error{
		Error: fmt.Sprintf("%s (%d)", e.Message, e.StatusCode),
		Type:  "error",
	}
	res, _ := json.Marshal(resObj)
	return res
}
