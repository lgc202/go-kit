package qwen

import "github.com/lgc202/go-kit/llm"

// 扩展字段键，用于 llm.WithExtraField()
const (
	extEnableThinking = "enable_thinking"
)

// WithThinking 启用或禁用深度思考模式
// 此参数仅对 Qwen 支持深度思考的模型有效
// 参考: https://www.alibabacloud.com/help/en/model-studio/deep-thinking
//
// - true: 启用深度思考模式
// - false: 禁用深度思考模式
func WithThinking(enabled bool) llm.RequestOption {
	return llm.WithExtraField(extEnableThinking, enabled)
}
