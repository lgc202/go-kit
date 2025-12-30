package llm

import (
	"context"

	"github.com/lgc202/go-kit/llm/schema"
)

// ChatModel is the minimal, provider-agnostic interface for chat-based LLMs.
type ChatModel interface {
	Chat(ctx context.Context, messages []schema.Message, opts ...RequestOption) (schema.ChatResponse, error)
	ChatStream(ctx context.Context, messages []schema.Message, opts ...RequestOption) (Stream, error)
}

// Stream is a provider-agnostic streaming reader.
//
// Recv returns (schema.StreamEvent, nil) for each event, and io.EOF when the
// stream ends normally.
type Stream interface {
	Recv() (schema.StreamEvent, error)
	Close() error
}

// Provider is the canonical identifier of a model provider.
type Provider string

const (
	ProviderUnknown  Provider = "unknown"
	ProviderOpenAI   Provider = "openai"
	ProviderDeepSeek Provider = "deepseek"
	ProviderKimi     Provider = "kimi"
	ProviderQwen     Provider = "qwen"
	ProviderOllama   Provider = "ollama"
)

// ProviderNamer is an optional interface for discovering which provider a
// ChatModel instance is backed by.
type ProviderNamer interface {
	Provider() Provider
}

func ProviderOf(m ChatModel) Provider {
	if p, ok := m.(ProviderNamer); ok {
		if p.Provider() != "" {
			return p.Provider()
		}
	}
	return ProviderUnknown
}
