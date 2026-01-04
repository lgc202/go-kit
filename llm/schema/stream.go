package schema

import "encoding/json"

// StreamOptions 配置流式响应的行为。
type StreamOptions struct {
	// IncludeUsage 表示是否在流式响应的最后一个块中包含使用统计信息。
	// 设置为 true 时，流式响应结束时会包含一个包含 usage 字段的块。
	IncludeUsage bool `json:"include_usage,omitempty"`
}

// StreamEventType 流式事件类型
type StreamEventType string

const (
	StreamEventDelta StreamEventType = "delta" // 增量内容
	StreamEventDone  StreamEventType = "done"  // 流结束
)

// StreamEvent 表示流式响应中的一个事件
type StreamEvent struct {
	Type        StreamEventType `json:"type"`
	ChoiceIndex int             `json:"choice_index,omitempty"`

	Delta        string        `json:"delta,omitempty"`
	Reasoning    string        `json:"reasoning,omitempty"`      // 推理内容，用于推理模型
	ToolCalls    []ToolCall    `json:"tool_calls,omitempty"`
	FinishReason *FinishReason `json:"finish_reason,omitempty"`
	Usage        *Usage        `json:"usage,omitempty"`

	// ExtraFields 是 provider 特定的扩展字段（通常由 stream event hook 从原始事件中提取并填充）
	ExtraFields map[string]any `json:"extra_fields,omitempty"`

	Raw json.RawMessage `json:"raw,omitempty"`
}
