package llm

import "fmt"

// UnsupportedOptionError indicates a request option or schema feature isn't
// supported by a given provider/client implementation.
//
// Use errors.As(err, *UnsupportedOptionError) to branch on this.
type UnsupportedOptionError struct {
	Provider Provider
	Option   string
	Reason   string
}

func (e *UnsupportedOptionError) Error() string {
	if e == nil {
		return "<nil>"
	}

	p := string(e.Provider)
	if p == "" {
		p = string(ProviderUnknown)
	}

	switch {
	case e.Option != "" && e.Reason != "":
		return fmt.Sprintf("%s: unsupported option %q: %s", p, e.Option, e.Reason)
	case e.Option != "":
		return fmt.Sprintf("%s: unsupported option %q", p, e.Option)
	case e.Reason != "":
		return fmt.Sprintf("%s: unsupported option: %s", p, e.Reason)
	default:
		return fmt.Sprintf("%s: unsupported option", p)
	}
}

// APIError represents a provider HTTP error with parsed fields when available.
//
// Providers typically return a JSON body with a message, type, and code. When
// those fields exist, openai_compat returns *APIError so callers can switch on
// StatusCode/Code/Type.
type APIError struct {
	Provider   string
	StatusCode int

	Message string
	Type    string
	Code    string

	// Body is the raw response body for non-JSON or unrecognized formats.
	Body []byte
}

func (e *APIError) Error() string {
	if e == nil {
		return "<nil>"
	}

	if e.Message != "" && e.Code != "" {
		return fmt.Sprintf("%s: http %d: %s (%s)", e.Provider, e.StatusCode, e.Message, e.Code)
	}
	if e.Message != "" {
		return fmt.Sprintf("%s: http %d: %s", e.Provider, e.StatusCode, e.Message)
	}
	if len(e.Body) > 0 {
		return fmt.Sprintf("%s: http %d: %s", e.Provider, e.StatusCode, string(e.Body))
	}
	return fmt.Sprintf("%s: http %d", e.Provider, e.StatusCode)
}
