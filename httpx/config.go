package httpx

import (
	"net/http"
	"time"
)

// Config configures a Client. Use DefaultConfig() as a baseline.
type Config struct {
	// BaseURL is optional. If set, relative paths passed to NewRequest are resolved against it.
	BaseURL string

	// Timeout sets an upper bound for the whole request including retries.
	// If the request context already has a deadline, the earlier one wins.
	Timeout time.Duration

	// Transport is the underlying RoundTripper. If nil, a tuned default is used.
	Transport http.RoundTripper

	// DefaultHeaders are copied into every request (caller headers win).
	DefaultHeaders http.Header

	// UserAgent is set when the request does not already have a User-Agent header.
	UserAgent string

	// Retry configures automatic retries.
	Retry RetryConfig

	// MaxErrorBodyBytes limits how many bytes are read into Error.RawBody for non-2xx responses.
	// If zero, DefaultMaxErrorBodyBytes is used.
	MaxErrorBodyBytes int64

	// RequestID configures correlation id propagation.
	RequestID RequestIDConfig
}

const DefaultMaxErrorBodyBytes int64 = 64 << 10 // 64KiB

// DefaultConfig returns a conservative baseline suitable for most services.
func DefaultConfig() Config {
	return Config{
		Timeout:           30 * time.Second,
		Transport:         DefaultTransport(),
		DefaultHeaders:    make(http.Header),
		UserAgent:         "",
		Retry:             DefaultRetryConfig(),
		MaxErrorBodyBytes: DefaultMaxErrorBodyBytes,
		RequestID:         DefaultRequestIDConfig(),
	}
}
