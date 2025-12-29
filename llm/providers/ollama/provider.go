package ollama

import "github.com/lgc202/go-kit/llm/providers/openai_compat"

const (
	DefaultBaseURL             = "http://localhost:11434"
	DefaultChatCompletionsPath = "/v1/chat/completions"
)

// New returns an Ollama provider via its OpenAI-compatible API.
//
// Requires Ollama to be running locally.
func New(apiKey string, opts ...Option) (*openai_compat.Provider, error) {
	return openai_compat.New(apiKey, append([]openai_compat.Option{
		openai_compat.WithProviderName("ollama"),
		openai_compat.WithBaseURL(DefaultBaseURL),
		openai_compat.WithChatCompletionsPath(DefaultChatCompletionsPath),
	}, opts...)...)
}
