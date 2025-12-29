package kimi

import "github.com/lgc202/go-kit/llm/providers/openai_compat"

const (
	DefaultBaseURL             = "https://api.moonshot.cn/v1"
	DefaultChatCompletionsPath = "/chat/completions"
)

// New returns a Kimi (Moonshot) provider.
//
// Kimi provides an OpenAI-compatible API.
func New(apiKey string, opts ...Option) (*openai_compat.Provider, error) {
	return openai_compat.New(apiKey, append([]openai_compat.Option{
		openai_compat.WithProviderName("kimi"),
		openai_compat.WithBaseURL(DefaultBaseURL),
		openai_compat.WithChatCompletionsPath(DefaultChatCompletionsPath),
	}, opts...)...)
}
