package obj

import (
	"encoding/json"
	"errors"
	"fmt"
)

var (
	HTTPErrorNotImplemented = HTTPError{StatusCode: 501, Message: "Not Implemented"}
)

type HTTPError struct {
	StatusCode int
	Message    string
}

func (e HTTPError) Error() string {
	return e.Message
}

func NewHTTPError(statusCode int, message string) *HTTPError {
	return &HTTPError{StatusCode: statusCode, Message: message}
}

func NewHTTPErrorf(statusCode int, format string, a ...interface{}) *HTTPError {
	message := fmt.Sprintf(format, a...)
	return &HTTPError{StatusCode: statusCode, Message: message}
}

func ErrorToHTTPError(statusCode int, err error) *HTTPError {
	if err == nil {
		return nil
	}
	var httpError HTTPError
	if errors.As(err, &httpError) {
		return &httpError
	}
	return &HTTPError{StatusCode: statusCode, Message: err.Error()}
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
