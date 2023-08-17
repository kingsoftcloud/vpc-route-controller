package http

import "fmt"

type Error struct {
	Code    int
	Message string
}

func (e *Error) Error() string {
	return fmt.Sprintf("http error: code %d, message %s", e.Code, e.Message)
}
