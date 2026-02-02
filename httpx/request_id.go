package httpx

import (
	"crypto/rand"
	"encoding/hex"
)

type RequestIDFunc func() string

type RequestIDConfig struct {
	// Header is the header name to carry the request id, e.g. "X-Request-ID".
	// If empty, request id injection is disabled.
	Header string

	// New generates a request id when the header is missing.
	// If nil, a default generator is used.
	New RequestIDFunc
}

func DefaultRequestIDConfig() RequestIDConfig {
	return RequestIDConfig{
		Header: "X-Request-ID",
		New:    DefaultRequestID,
	}
}

func DefaultRequestID() string {
	var b [16]byte
	if _, err := rand.Read(b[:]); err != nil {
		// Extremely unlikely; return empty rather than panicking.
		return ""
	}
	return hex.EncodeToString(b[:])
}
