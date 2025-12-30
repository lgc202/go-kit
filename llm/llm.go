package llm

import (
	"context"

	"github.com/lgc202/go-kit/llm/schema"
)

// ChatModel 是用于基于聊天的 LLM 的最小化、提供商无关接口
type ChatModel interface {
	Chat(ctx context.Context, messages []schema.Message, opts ...RequestOption) (schema.ChatResponse, error)
	ChatStream(ctx context.Context, messages []schema.Message, opts ...RequestOption) (Stream, error)
}

// Stream 是提供商无关的流式读取器
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

// ProviderNamer 是一个可选接口，用于发现 ChatModel 实例背后的提供商
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
