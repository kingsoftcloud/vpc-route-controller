package util

import (
	"fmt"
)

type ErrorResponse struct {
	StatusCode int //Status Code of HTTP Response
	Message    string
}

// An Error represents a custom error for Appengine API failure response
type Error struct {
	KopError ErrorResponse `json:"apperror"`
}

func (e *Error) Error() string {
	return fmt.Sprintf("Kop Error: Status Code: %d Message: %s", e.KopError.StatusCode, e.KopError.Message)
}
