package llm

import (
	"errors"
	"fmt"
)

type ErrorKind string

const (
	ErrKindAuth       ErrorKind = "auth"
	ErrKindRateLimit  ErrorKind = "rate_limit"
	ErrKindBadRequest ErrorKind = "bad_request"
	ErrKindNotFound   ErrorKind = "not_found"
	ErrKindServer     ErrorKind = "server"
	ErrKindTimeout    ErrorKind = "timeout"
	ErrKindCanceled   ErrorKind = "canceled"
	ErrKindParse      ErrorKind = "parse"
	ErrKindUnknown    ErrorKind = "unknown"
)

// LLMError is a provider-agnostic error container.
//
// It is designed for enterprise use: stable classification, raw payload access,
// and retry-related hints.
type LLMError struct {
	Provider string
	Kind     ErrorKind

	HTTPStatus   int
	ProviderCode string
	Message      string

	Retryable bool

	// Raw is an optional raw error payload (e.g. the HTTP response body).
	Raw []byte

	Cause error
}

func (e *LLMError) Error() string {
	msg := e.Message
	if msg == "" {
		msg = string(e.Kind)
	}
	if e.Provider != "" {
		return fmt.Sprintf("llm %s: %s", e.Provider, msg)
	}
	return fmt.Sprintf("llm: %s", msg)
}

func (e *LLMError) Unwrap() error { return e.Cause }

func AsLLMError(err error) (*LLMError, bool) {
	var e *LLMError
	if errors.As(err, &e) {
		return e, true
	}
	return nil, false
}
