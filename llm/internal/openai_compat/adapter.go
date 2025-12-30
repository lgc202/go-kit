package openai_compat

import (
	"encoding/json"

	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/schema"
)

// Adapter 定制 OpenAI 兼容的 ChatCompletions 方言差异
//
// 大多数提供商可以使用 NoopAdapter，只有请求/响应存在差异的
// 提供商（如额外的请求字段）需要自定义 Adapter
type Adapter interface {
	ApplyRequestExtensions(req map[string]any, cfg llm.RequestConfig) error
	ParseError(provider string, statusCode int, body []byte) error

	// 可选钩子，兼容核心已经提取了通用字段（content、reasoning_content、tool_calls）
	// 提供商可以根据需要进一步丰富这些字段
	EnrichResponseMessage(dst *schema.Message, rawMessage json.RawMessage) error
	EnrichStreamDelta(dst *schema.StreamEvent, rawDelta json.RawMessage) error
}

type NoopAdapter struct{}

func (NoopAdapter) ApplyRequestExtensions(_ map[string]any, _ llm.RequestConfig) error { return nil }
func (NoopAdapter) ParseError(_ string, _ int, _ []byte) error                         { return nil }
func (NoopAdapter) EnrichResponseMessage(_ *schema.Message, _ json.RawMessage) error   { return nil }
func (NoopAdapter) EnrichStreamDelta(_ *schema.StreamEvent, _ json.RawMessage) error   { return nil }
