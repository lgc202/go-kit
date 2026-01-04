package llm

import (
	"context"

	"github.com/lgc202/go-kit/llm/schema"
)

// ChatModel 聊天模型接口，最小化且与具体 provider 无关
type ChatModel interface {
	Chat(ctx context.Context, messages []schema.Message, opts ...ChatOption) (schema.ChatResponse, error)
	ChatStream(ctx context.Context, messages []schema.Message, opts ...ChatOption) (Stream, error)
}

// Embedder 向量嵌入接口，与具体 provider 无关
type Embedder interface {
	Embed(ctx context.Context, inputs []string, opts ...EmbeddingOption) (schema.EmbeddingResponse, error)
}

// Stream 流式响应读取器
//
// Recv 每次返回一个事件，流结束时返回 io.EOF
type Stream interface {
	Recv() (schema.StreamEvent, error)
	Close() error
}

// Provider 模型提供商标识
type Provider string

const (
	ProviderUnknown  Provider = "unknown"
	ProviderOpenAI   Provider = "openai"
	ProviderDeepSeek Provider = "deepseek"
	ProviderKimi     Provider = "kimi"
	ProviderQwen     Provider = "qwen"
	ProviderOllama   Provider = "ollama"
)

// ProviderNamer 可选接口，用于标识 ChatModel 的 provider 类型
type ProviderNamer interface {
	Provider() Provider
}

// ProviderOf 获取 ChatModel 的 provider 标识
func ProviderOf(m ChatModel) Provider {
	if p, ok := m.(ProviderNamer); ok {
		if p.Provider() != "" {
			return p.Provider()
		}
	}
	return ProviderUnknown
}
