package chat

import "github.com/lgc202/go-kit/llm"

// 扩展字段键，用于 llm.WithExtraField()
const (
	extEnableThinking = "enable_thinking"
)

// WithThinking 启用或禁用深度思考模式
// 此参数仅对 Qwen 支持深度思考的模型有效
// true: 启用深度思考模式
// false: 禁用深度思考模式
func WithThinking(enabled bool) llm.ChatOption {
	return llm.WithExtraField(extEnableThinking, enabled)
}
