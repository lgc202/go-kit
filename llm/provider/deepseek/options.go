package deepseek

import "github.com/lgc202/go-kit/llm"

// WithThinking 启用或禁用推理模式。
// - true: 启用推理（deepseek-reasoner 默认值）
// - false: 禁用推理
func WithThinking(enabled bool) llm.RequestOption {
	if enabled {
		return llm.WithExtraField("thinking", map[string]string{"type": "enabled"})
	}
	return llm.WithExtraField("thinking", map[string]string{"type": "disabled"})
}
