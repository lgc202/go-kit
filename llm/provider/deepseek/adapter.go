package deepseek

import (
	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/internal/openai_compat"
)

type adapter struct {
	openai_compat.NoopAdapter
}

// ApplyRequestExtensions 应用 DeepSeek 特有的扩展到请求。
// 只处理标准 OpenAI 兼容层不支持的字段。
func (adapter) ApplyRequestExtensions(req map[string]any, cfg llm.RequestConfig) error {
	if cfg.ExtraFields == nil {
		return nil
	}

	// thinking - DeepSeek 特有的推理控制
	if v, ok := cfg.ExtraFields[ExtThinking]; ok {
		req["thinking"] = v
	}

	return nil
}
