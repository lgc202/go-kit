package chat

import "github.com/lgc202/go-kit/llm"

// 扩展字段键，用于 llm.WithExtraField()
const (
	extFormat    = "format"
	extKeepAlive = "keep_alive"
	extOptions   = "options"
	extThink     = "think"
)

// WithFormat 设置结构化输出格式，传入 JSON Schema 以启用 JSON 模式
func WithFormat(jsonSchema map[string]any) llm.ChatOption {
	return llm.WithExtraField(extFormat, jsonSchema)
}

// WithKeepAlive 设置模型在内存中保持加载的时间
// 例如 "5m" (5分钟), "24h" (24小时)，为空则使用默认值
func WithKeepAlive(duration string) llm.ChatOption {
	if duration == "" {
		return llm.WithExtraField(extKeepAlive, nil)
	}
	return llm.WithExtraField(extKeepAlive, duration)
}

// WithOptions 设置 Ollama 模型运行选项
// 这些选项直接传递给 Ollama 的 options 参数
// 常用选项包括: temperature, top_k, top_p, num_ctx, num_predict, repeat_penalty, stop 等
func WithOptions(options map[string]any) llm.ChatOption {
	return llm.WithExtraField(extOptions, options)
}

// WithThink 启用推理模式，用于支持推理的 Ollama 模型
func WithThink(enabled bool) llm.ChatOption {
	return llm.WithExtraField(extThink, enabled)
}
