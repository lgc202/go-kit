package deepseek

import (
	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/providers/openai_compat"
)

// Option configures the DeepSeek provider (HTTP / endpoint behavior).
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

// WithThinking sets the DeepSeek `thinking` field.
//
// The same option can be used:
// - as a client default: llm.New(provider, deepseek.WithThinkingDisabled())
// - per call override:  client.Chat(ctx, msgs, deepseek.WithThinkingEnabled())
func WithThinking(cfg Thinking) llm.RequestOption {
	return llm.WithExtra("thinking", cfg)
}

func WithThinkingDisabled() llm.RequestOption { return WithThinking(Thinking{Type: ThinkingDisabled}) }
func WithThinkingEnabled() llm.RequestOption  { return WithThinking(Thinking{Type: ThinkingEnabled}) }
