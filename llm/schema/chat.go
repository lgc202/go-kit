package schema

import (
	"encoding/json"
	"time"
)

type FinishReason string

const (
	FinishReasonStop          FinishReason = "stop"
	FinishReasonLength        FinishReason = "length"
	FinishReasonToolCalls     FinishReason = "tool_calls"
	FinishReasonContentFilter FinishReason = "content_filter"
)

type Choice struct {
	Index        int          `json:"index"`
	Message      Message      `json:"message"`
	FinishReason FinishReason `json:"finish_reason"`
}

type ChatResponse struct {
	ID        string    `json:"id"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`

	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`

	// ServiceTier 返回实际使用的服务层级
	ServiceTier *string `json:"service_tier,omitempty"`

	// ExtraFields 是 provider 特定的扩展字段（通常由 response hook 从原始响应中提取并填充）。
	ExtraFields map[string]any `json:"extra_fields,omitempty"`

	// Raw 保留 provider 原生载荷，用于调试/向前兼容
	Raw json.RawMessage `json:"raw,omitempty"`
}
