package openai_compat

import (
	"encoding/json"

	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/schema"
)

// Adapter customizes OpenAI-compatible ChatCompletions dialect differences.
//
// Most providers can use NoopAdapter; only providers with request/response
// divergences (e.g. extra request fields) need a custom Adapter.
type Adapter interface {
	ApplyRequestExtensions(req map[string]any, cfg llm.RequestConfig) error
	ParseError(provider string, statusCode int, body []byte) error

	// Optional hooks. The compat core already extracts common fields (content,
	// reasoning_content, tool_calls). Providers can enrich further if needed.
	EnrichResponseMessage(dst *schema.Message, rawMessage json.RawMessage) error
	EnrichStreamDelta(dst *schema.StreamEvent, rawDelta json.RawMessage) error
}

type NoopAdapter struct{}

func (NoopAdapter) ApplyRequestExtensions(_ map[string]any, _ llm.RequestConfig) error { return nil }
func (NoopAdapter) ParseError(_ string, _ int, _ []byte) error                         { return nil }
func (NoopAdapter) EnrichResponseMessage(_ *schema.Message, _ json.RawMessage) error   { return nil }
func (NoopAdapter) EnrichStreamDelta(_ *schema.StreamEvent, _ json.RawMessage) error   { return nil }
