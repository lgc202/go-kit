package llm

import (
	"context"

	"github.com/lgc202/go-kit/llm/schema"
)

// ChatModel 是用于 chat 的 LLM 的最小化、provider 无关接口
type ChatModel interface {
	Chat(ctx context.Context, messages []schema.Message, opts ...ChatOption) (schema.ChatResponse, error)
	ChatStream(ctx context.Context, messages []schema.Message, opts ...ChatOption) (Stream, error)
}

// Embedder is the minimal, provider-agnostic interface for text embeddings.
type Embedder interface {
	Embed(ctx context.Context, inputs []string, opts ...EmbeddingOption) (schema.EmbeddingResponse, error)
}

// Stream 是 provider 无关的流式读取器
//
// Recv 为每个事件返回 (schema.StreamEvent, nil)，当流正常结束时返回 io.EOF
type Stream interface {
	Recv() (schema.StreamEvent, error)
	Close() error
}

// Provider 是模型提供商的标准标识符
type Provider string

const (
	ProviderUnknown  Provider = "unknown"
	ProviderOpenAI   Provider = "openai"
	ProviderDeepSeek Provider = "deepseek"
	ProviderKimi     Provider = "kimi"
	ProviderQwen     Provider = "qwen"
	ProviderOllama   Provider = "ollama"
)

// ProviderNamer 是一个可选接口，用于发现 ChatModel 实例背后的 provider
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
