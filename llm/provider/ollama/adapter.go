package ollama

import (
	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/internal/openai_compat"
)

type adapter struct {
	openai_compat.NoopAdapter
}

var _ openai_compat.Adapter = adapter{}

// ApplyRequestExtensions 应用 Ollama 特有的扩展到请求
// 只处理标准 OpenAI 兼容层不支持的字段
func (adapter) ApplyRequestExtensions(req map[string]any, cfg llm.RequestConfig) error {
	if cfg.ExtraFields == nil {
		return nil
	}

	// format - JSON 结构化输出
	if v, ok := cfg.ExtraFields[extFormat]; ok {
		req["format"] = v
	}

	// keep_alive - 模型保持加载时间
	if v, ok := cfg.ExtraFields[extKeepAlive]; ok {
		req["keep_alive"] = v
	}

	// options - Ollama 模型运行选项
	if v, ok := cfg.ExtraFields[extOptions]; ok {
		req["options"] = v
	}

	// think - 推理模式
	if v, ok := cfg.ExtraFields[extThink]; ok {
		req["think"] = v
	}

	return nil
}
