package qwen

import "github.com/lgc202/go-kit/llm/providers/openai_compat"

const (
	DefaultBaseURL             = "https://dashscope.aliyuncs.com/compatible-mode/v1"
	DefaultChatCompletionsPath = "/chat/completions"
)

// New returns a Qwen (DashScope) provider using OpenAI-compatible mode.
func New(apiKey string, opts ...Option) (*openai_compat.Provider, error) {
	return openai_compat.New(apiKey, append([]openai_compat.Option{
		openai_compat.WithProviderName("qwen"),
		openai_compat.WithBaseURL(DefaultBaseURL),
		openai_compat.WithChatCompletionsPath(DefaultChatCompletionsPath),
	}, opts...)...)
}
