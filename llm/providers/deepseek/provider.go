package deepseek

import "github.com/lgc202/go-kit/llm/providers/openai_compat"

const (
	DefaultBaseURL             = "https://api.deepseek.com"
	DefaultChatCompletionsPath = "/chat/completions"
)

// New returns a DeepSeek provider.
//
// DeepSeek advertises OpenAI-compatibility, but differences can be handled
// through openai_compat hooks/options.
func New(apiKey string, opts ...Option) (*openai_compat.Provider, error) {
	return openai_compat.New(apiKey, append([]openai_compat.Option{
		openai_compat.WithProviderName("deepseek"),
		openai_compat.WithBaseURL(DefaultBaseURL),
		openai_compat.WithChatCompletionsPath(DefaultChatCompletionsPath),
	}, opts...)...)
}
