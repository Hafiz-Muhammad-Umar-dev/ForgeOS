package query

import "errors"

var (
	ErrIntentNotFound = errors.New("query: intent not found")
	ErrTaskNotFound   = errors.New("query: task not found")
)
