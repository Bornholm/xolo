package common

import (
	"net/http"

	"github.com/bornholm/xolo/internal/http/handler/webui/common/component"
)

type Error struct {
	err         string
	userMessage string
	statusCode  int
	links       []component.LinkItem
}

// Links implements [WithErrorLinks].
func (e *Error) Links() []component.LinkItem {
	return e.links
}

// StatusCode implements HTTPError.
func (e *Error) StatusCode() int {
	return e.statusCode
}

// Error implements UserFacingError.
func (e *Error) Error() string {
	return e.err
}

// UserMessage implements UserFacingError.
func (e *Error) UserMessage() string {
	return e.userMessage
}

func NewError(err string, userMessage string, statusCode int, links ...component.LinkItem) *Error {
	return &Error{err, userMessage, statusCode, links}
}

var _ UserFacingError = &Error{}
var _ HTTPError = &Error{}
var _ WithErrorLinks = &Error{}

func NewHTTPError(statusCode int, links ...component.LinkItem) *Error {
	return &Error{http.StatusText(statusCode), http.StatusText(statusCode), statusCode, links}
}
