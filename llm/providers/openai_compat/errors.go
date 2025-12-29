package openai_compat

import (
	"context"
	"encoding/json"
	"errors"
	"net/http"

	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/internal/transport"
)

func (p *Provider) mapError(err error, raw []byte) error {
	if errors.Is(err, context.Canceled) {
		return &llm.LLMError{Provider: p.name, Kind: llm.ErrKindCanceled, Message: "request canceled", Cause: err}
	}
	if errors.Is(err, context.DeadlineExceeded) {
		return &llm.LLMError{Provider: p.name, Kind: llm.ErrKindTimeout, Message: "request deadline exceeded", Retryable: true, Cause: err}
	}

	var se *transport.HTTPStatusError
	if errors.As(err, &se) {
		kind, retryable := classifyHTTP(se.StatusCode)
		msg, code := parseErrorEnvelope(se.Body)
		if msg == "" {
			msg = http.StatusText(se.StatusCode)
		}
		return &llm.LLMError{
			Provider:     p.name,
			Kind:         kind,
			HTTPStatus:   se.StatusCode,
			ProviderCode: code,
			Message:      msg,
			Retryable:    retryable,
			Raw:          append([]byte(nil), se.Body...),
			Cause:        err,
		}
	}

	return &llm.LLMError{Provider: p.name, Kind: llm.ErrKindUnknown, Message: err.Error(), Retryable: true, Raw: raw, Cause: err}
}

func classifyHTTP(status int) (llm.ErrorKind, bool) {
	switch status {
	case http.StatusUnauthorized, http.StatusForbidden:
		return llm.ErrKindAuth, false
	case http.StatusTooManyRequests:
		return llm.ErrKindRateLimit, true
	case http.StatusBadRequest:
		return llm.ErrKindBadRequest, false
	case http.StatusNotFound:
		return llm.ErrKindNotFound, false
	case http.StatusRequestTimeout:
		return llm.ErrKindTimeout, true
	default:
		if status >= 500 {
			return llm.ErrKindServer, true
		}
		return llm.ErrKindUnknown, false
	}
}

func parseErrorEnvelope(raw []byte) (message string, code string) {
	var env errorEnvelope
	if err := json.Unmarshal(raw, &env); err != nil || env.Error == nil {
		return "", ""
	}
	if env.Error.Message != "" {
		message = env.Error.Message
	}
	if env.Error.Code != nil {
		code = stringify(env.Error.Code)
	}
	return message, code
}

func stringify(v any) string {
	switch x := v.(type) {
	case string:
		return x
	default:
		b, _ := json.Marshal(x)
		return string(b)
	}
}

func firstNonEmpty(vs ...string) string {
	for _, v := range vs {
		if v != "" {
			return v
		}
	}
	return ""
}
