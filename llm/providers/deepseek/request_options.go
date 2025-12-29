package deepseek

import "github.com/lgc202/go-kit/llm"

// WithThinking sets the DeepSeek `thinking` field for a single request.
//
// Use it per request:
//
//	llm.NewChatRequest(...).With(deepseek.WithThinking(...))
//
// Or as a provider default:
//
//	deepseek.New(key, deepseek.WithDefaultRequest(deepseek.WithThinking(...)))
func WithThinking(cfg Thinking) llm.RequestOption {
	return llm.WithExtra("thinking", cfg)
}

func WithThinkingDisabled() llm.RequestOption { return WithThinking(Thinking{Type: ThinkingDisabled}) }
func WithThinkingEnabled() llm.RequestOption  { return WithThinking(Thinking{Type: ThinkingEnabled}) }
