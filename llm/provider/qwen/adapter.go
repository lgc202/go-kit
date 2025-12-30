package qwen

import (
	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/internal/openai_compat"
)

type adapter struct {
	openai_compat.NoopAdapter
}

var _ openai_compat.Adapter = adapter{}

// ApplyRequestExtensions 应用 Qwen 特有的扩展到请求
// 只处理标准 OpenAI 兼容层不支持的字段
func (adapter) ApplyRequestExtensions(req map[string]any, cfg llm.RequestConfig) error {
	if cfg.ExtraFields == nil {
		return nil
	}

	// enable_thinking - Qwen 深度思考模式
	// 支持 vLLM 和 Bailian 两种方式
	if v, ok := cfg.ExtraFields[extEnableThinking]; ok {
		req["enable_thinking"] = v
	}

	return nil
}
