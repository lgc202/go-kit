package httpx

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// Error represents an HTTP or transport error with observability-friendly fields.
type Error struct {
	Method string
	URL    string

	// StatusCode is the HTTP status code. It is 0 when the request failed before receiving a response.
	StatusCode int

	// RequestID is extracted from the configured RequestID header (see RequestIDConfig).
	RequestID string

	// RetryAfter is parsed from Retry-After when present.
	RetryAfter time.Duration

	// RawBody is a truncated copy of the response body (only for non-2xx responses).
	RawBody []byte

	// Cause is the underlying error (transport error, context cancellation, JSON decode error, etc).
	Cause error

	// Retryable indicates whether the error is likely safe to retry (policy dependent).
	Retryable bool
}

func (e *Error) Error() string {
	if e == nil {
		return "<nil>"
	}
	var b strings.Builder
	if strings.TrimSpace(e.Method) != "" {
		b.WriteString(strings.ToUpper(strings.TrimSpace(e.Method)))
		b.WriteString(" ")
	}
	if strings.TrimSpace(e.URL) != "" {
		b.WriteString(strings.TrimSpace(e.URL))
		b.WriteString(": ")
	}
	if e.StatusCode != 0 {
		b.WriteString(fmt.Sprintf("http %d", e.StatusCode))
		if t := strings.TrimSpace(http.StatusText(e.StatusCode)); t != "" {
			b.WriteString(" ")
			b.WriteString(t)
		}
	} else {
		b.WriteString("request failed")
	}
	if e.RequestID != "" {
		b.WriteString(" request_id=")
		b.WriteString(e.RequestID)
	}
	if e.Cause != nil {
		b.WriteString(": ")
		b.WriteString(e.Cause.Error())
	}
	return b.String()
}

func (e *Error) Unwrap() error { return e.Cause }

// AsError extracts *Error.
func AsError(err error) (*Error, bool) {
	var he *Error
	if errors.As(err, &he) {
		return he, true
	}
	return nil, false
}

func IsRetryable(err error) bool {
	he, ok := AsError(err)
	return ok && he.Retryable
}

func IsHTTPStatus(err error, code int) bool {
	he, ok := AsError(err)
	return ok && he.StatusCode == code
}
