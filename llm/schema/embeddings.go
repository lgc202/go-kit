package schema

import (
	"encoding/json"
)

type Embedding struct {
	Index  int       `json:"index"`
	Vector []float64 `json:"vector"`
}

type EmbeddingResponse struct {
	Model string `json:"model"`

	Data  []Embedding `json:"data"`
	Usage Usage       `json:"usage"`

	ExtraFields map[string]any  `json:"extra_fields,omitempty"`
	Raw         json.RawMessage `json:"raw,omitempty"`
}
