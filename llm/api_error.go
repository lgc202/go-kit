package llm

import (
	"errors"
	"fmt"
	"net/http"
	"strings"
	"time"
)

// APIError is a provider-agnostic error returned for non-2xx API responses.
//
// It is designed for enterprise-style handling: classification (rate limit, auth),
// observability (request id), and backoff (retry-after) without provider branches.
type APIError struct {
	Provider   Provider
	StatusCode int

	// Code is the provider-specific error code (when available).
	Code string

	// Type is the provider-specific error type (when available).
	Type string

	// Message is the human-readable error message (when available).
	Message string

	// RequestID is a provider-specific request identifier, usually extracted from HTTP headers.
	RequestID string

	// RetryAfter is the parsed Retry-After header (when present).
	RetryAfter time.Duration

	// Raw is the raw response body bytes for debugging/forward-compat.
	Raw []byte
}

func (e *APIError) Error() string {
	if e == nil {
		return "<nil>"
	}

	var b strings.Builder
	if strings.TrimSpace(string(e.Provider)) != "" {
		b.WriteString(string(e.Provider))
		b.WriteString(": ")
	}
	if e.StatusCode != 0 {
		b.WriteString(fmt.Sprintf("http %d", e.StatusCode))
	} else {
		b.WriteString("http error")
	}

	msg := strings.TrimSpace(e.Message)
	if msg == "" && e.StatusCode != 0 {
		msg = http.StatusText(e.StatusCode)
	}
	if msg != "" {
		b.WriteString(": ")
		b.WriteString(msg)
	}

	if strings.TrimSpace(e.Code) != "" {
		b.WriteString(" (")
		b.WriteString(strings.TrimSpace(e.Code))
		b.WriteString(")")
	}
	if strings.TrimSpace(e.RequestID) != "" {
		b.WriteString(" request_id=")
		b.WriteString(strings.TrimSpace(e.RequestID))
	}

	return b.String()
}

func AsAPIError(err error) (*APIError, bool) {
	var ae *APIError
	if errors.As(err, &ae) {
		return ae, true
	}
	return nil, false
}

func IsRateLimit(err error) bool {
	ae, ok := AsAPIError(err)
	if !ok {
		return false
	}
	if ae.StatusCode == http.StatusTooManyRequests {
		return true
	}
	code := strings.ToLower(strings.TrimSpace(ae.Code))
	return code == "rate_limit" || code == "rate_limit_exceeded"
}

func IsAuth(err error) bool {
	ae, ok := AsAPIError(err)
	if !ok {
		return false
	}
	return ae.StatusCode == http.StatusUnauthorized || ae.StatusCode == http.StatusForbidden
}

func IsTemporary(err error) bool {
	ae, ok := AsAPIError(err)
	if !ok {
		return false
	}
	switch ae.StatusCode {
	case http.StatusRequestTimeout, http.StatusTooManyRequests:
		return true
	case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
		return true
	default:
		return false
	}
}
