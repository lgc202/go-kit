package openai_compat

import (
	"encoding/json"

	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/schema"
)

// Adapter 定制 OpenAI 兼容的 ChatCompletions 差异
//
// 大多数 provider 可以使用 NoopAdapter，只有请求/响应存在差异的
// Provider（如额外的请求字段）需要自定义 Adapter，如果不想实现所有接口，可以先嵌入 NoopAdapter，如
//
//	type adapter struct {
//		openai_compat.NoopAdapter
//	}
type Adapter interface {
	ApplyRequestExtensions(req *chatCompletionRequest, cfg llm.RequestConfig) error
	ParseError(provider string, statusCode int, body []byte) error

	// EnrichResponse 丰富响应数据（处理特有字段如 reasoning_content、reasoning_tokens 等）
	// 兼容核心已经提取了通用字段，provider 可以进一步处理原始响应
	EnrichResponse(dst *schema.ChatResponse, rawResponse json.RawMessage) error

	// EnrichStreamEvent 丰富流式响应事件
	EnrichStreamEvent(dst *schema.StreamEvent, rawEvent json.RawMessage) error
}

// NoopAdapter 空操作适配器，适用于标准 OpenAI 兼容 provider
type NoopAdapter struct{}

func (NoopAdapter) ApplyRequestExtensions(_ *chatCompletionRequest, _ llm.RequestConfig) error {
	return nil
}
func (NoopAdapter) ParseError(_ string, _ int, _ []byte) error                       { return nil }
func (NoopAdapter) EnrichResponse(_ *schema.ChatResponse, _ json.RawMessage) error   { return nil }
func (NoopAdapter) EnrichStreamEvent(_ *schema.StreamEvent, _ json.RawMessage) error { return nil }
