// package: obj / HTTP error type
// type:    data
// job:     defines HTTPError carrying an HTTP status code, error code, and message
// limits:  error value and JSON rendering only; no HTTP response writing (-> api/httpx)
package obj

import (
	"encoding/json"
	"fmt"
)

var (
	HTTPErrorNotImplemented = HTTPError{StatusCode: 501, Message: "Not Implemented"}
)

// HTTPError is an error carrying an HTTP status code, frontend error code, and message.
type HTTPError struct {
	StatusCode int
	Code       string // Machine-readable error code for frontend
	Message    string
}

// Error implements the error interface, returning the message.
func (e HTTPError) Error() string {
	return e.Message
}

// NewHTTPErrorWithCode creates an HTTPError with the given status code, error code, and message.
func NewHTTPErrorWithCode(statusCode int, code string, message string) *HTTPError {
	return &HTTPError{StatusCode: statusCode, Code: code, Message: message}
}

// Json renders the error as a JSON error response body.
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
