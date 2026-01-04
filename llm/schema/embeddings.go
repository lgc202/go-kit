package schema

import (
	"encoding/json"
)

// Embedding 表示单个文本嵌入向量
type Embedding struct {
	Index  int       `json:"index"`
	Vector []float64 `json:"vector"`
}

// EmbeddingResponse 表示嵌入向量响应
type EmbeddingResponse struct {
	Model string `json:"model"`

	Data  []Embedding `json:"data"`
	Usage Usage       `json:"usage"`

	ExtraFields map[string]any  `json:"extra_fields,omitempty"` // provider 特定的扩展字段
	Raw         json.RawMessage `json:"raw,omitempty"`           // 原始响应
}
