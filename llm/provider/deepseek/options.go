package deepseek

import (
	"github.com/lgc202/go-kit/llm"
)

// 扩展字段键，用于 llm.WithExtraField()。
const (
	ExtThinking = "thinking"
)

// ThinkingType 控制推理模型的推理行为。
type ThinkingType string

const (
	ThinkingTypeEnabled  ThinkingType = "enabled"
	ThinkingTypeDisabled ThinkingType = "disabled"
)

// Thinking 控制推理模型的推理行为。
type Thinking struct {
	Type ThinkingType `json:"type"`
}

// WithThinking 启用或禁用推理模式。
// - true: 启用推理（deepseek-reasoner 默认值）
// - false: 禁用推理
func WithThinking(enabled bool) llm.RequestOption {
	if enabled {
		return llm.WithExtraField(ExtThinking, Thinking{Type: ThinkingTypeEnabled})
	}
	return llm.WithExtraField(ExtThinking, Thinking{Type: ThinkingTypeDisabled})
}
