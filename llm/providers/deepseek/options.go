package deepseek

import "github.com/lgc202/go-kit/llm/providers/openai_compat"

// Option configures the DeepSeek provider (client-level defaults).
type Option = openai_compat.Option

// Re-export common OpenAI-compatible options.
var (
	WithBaseURL             = openai_compat.WithBaseURL
	WithHTTPClient          = openai_compat.WithHTTPClient
	WithUserAgent           = openai_compat.WithUserAgent
	WithLogger              = openai_compat.WithLogger
	WithRetry               = openai_compat.WithRetry
	WithDefaultHeader       = openai_compat.WithDefaultHeader
	WithChatCompletionsPath = openai_compat.WithChatCompletionsPath
	WithDefaultModel        = openai_compat.WithDefaultModel
	WithDefaultRequest      = openai_compat.WithDefaultRequest
	WithHooks               = openai_compat.WithHooks
)

type ThinkingType string

const (
	ThinkingDisabled ThinkingType = "disabled"
	ThinkingEnabled  ThinkingType = "enabled"
)

// Thinking controls DeepSeek request field `thinking`.
//
// Example JSON:
//
//	{"thinking": {"type": "disabled"}}
type Thinking struct {
	Type ThinkingType `json:"type"`
}

// WithDefaultThinking sets a provider-level default for DeepSeek `thinking`.
//
// Equivalent to:
//
//	WithDefaultRequest(deepseek.WithThinking(cfg))
func WithDefaultThinking(cfg Thinking) Option {
	return WithDefaultRequest(WithThinking(cfg))
}

func WithDefaultThinkingDisabled() Option {
	return WithDefaultThinking(Thinking{Type: ThinkingDisabled})
}
func WithDefaultThinkingEnabled() Option { return WithDefaultThinking(Thinking{Type: ThinkingEnabled}) }
