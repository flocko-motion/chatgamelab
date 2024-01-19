package obj

import (
	"encoding/json"
	"errors"
	"fmt"
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
		Error   int    `json:"error"`
		Message string `json:"message"`
	}
	resObj := Error{
		Message: e.Message,
		Error:   0,
	}
	if e.StatusCode != 200 {
		resObj.Error = e.StatusCode
	}
	res, _ := json.Marshal(resObj)
	return res
}
