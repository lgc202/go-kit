package openai_compat

import (
	"net/http"

	"github.com/lgc202/go-kit/llm"
)

type Option func(*Provider) error

type Hooks struct {
	PatchHeaders func(h http.Header)

	// BeforeMap allows mutating the canonical request before mapping.
	// Implementations should only touch req.Extra and similar optional fields.
	BeforeMap func(req *llm.ChatRequest)

	// PatchRequest allows mutating the final JSON request map.
	// This is the primary escape hatch for "OpenAI-compatible" providers.
	PatchRequest func(m map[string]any)
}

func WithHooks(h Hooks) Option {
	return func(p *Provider) error {
		prev := p.hooks
		p.hooks.PatchHeaders = chainHeaders(prev.PatchHeaders, h.PatchHeaders)
		p.hooks.BeforeMap = chainBeforeMap(prev.BeforeMap, h.BeforeMap)
		p.hooks.PatchRequest = chainPatchRequest(prev.PatchRequest, h.PatchRequest)
		return nil
	}
}

// WithDefaultRequest applies request-level options to every request sent by this provider.
//
// This lets you reuse the same llm.RequestOption both:
// - as a client-level default (via provider.New(..., WithDefaultRequest(...)))
// - per request (via req.With(...))
func WithDefaultRequest(opts ...llm.RequestOption) Option {
	return WithHooks(Hooks{
		BeforeMap: func(req *llm.ChatRequest) {
			for _, opt := range opts {
				if opt != nil {
					opt(req)
				}
			}
		},
	})
}

func chainHeaders(a, b func(http.Header)) func(http.Header) {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	return func(h http.Header) {
		a(h)
		b(h)
	}
}

func chainBeforeMap(a, b func(*llm.ChatRequest)) func(*llm.ChatRequest) {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	return func(r *llm.ChatRequest) {
		a(r)
		b(r)
	}
}

func chainPatchRequest(a, b func(map[string]any)) func(map[string]any) {
	if a == nil {
		return b
	}
	if b == nil {
		return a
	}
	return func(m map[string]any) {
		a(m)
		b(m)
	}
}
