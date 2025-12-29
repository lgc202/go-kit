package openai

import "github.com/lgc202/go-kit/llm/providers/openai_compat"

const DefaultBaseURL = "https://api.openai.com"

// New returns an OpenAI provider using the Chat Completions endpoint.
func New(apiKey string, opts ...Option) (*openai_compat.Provider, error) {
	return openai_compat.New(apiKey, append([]openai_compat.Option{
		openai_compat.WithProviderName("openai"),
		openai_compat.WithBaseURL(DefaultBaseURL),
	}, opts...)...)
}
