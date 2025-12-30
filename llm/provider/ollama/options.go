package ollama

import "github.com/lgc202/go-kit/llm"

// 扩展字段键，用于 llm.WithExtraField()
const (
	extFormat     = "format"
	extKeepAlive  = "keep_alive"
	extOptions    = "options"
	extThink      = "think"
)

// WithFormat 设置结构化输出格式
// 传入 JSON Schema 以启用 JSON 模式
func WithFormat(jsonSchema map[string]any) llm.RequestOption {
	return llm.WithExtraField(extFormat, jsonSchema)
}

// WithKeepAlive 设置模型在内存中保持加载的时间
// duration: 例如 "5m" (5分钟), "24h" (24小时)
// 如果为空，则使用默认值
func WithKeepAlive(duration string) llm.RequestOption {
	if duration == "" {
		return llm.WithExtraField(extKeepAlive, nil)
	}
	return llm.WithExtraField(extKeepAlive, duration)
}

// WithOptions 设置 Ollama 模型运行选项
// 这些选项直接传递给 Ollama 的 options 参数
// 参考: https://github.com/ollama/ollama/blob/main/docs/api.md#parameters-1
//
// 常用选项包括:
//   - temperature: 采样温度
//   - top_k: 限制候选 token 数量
//   - top_p: 核采样阈值
//   - num_ctx: 上下文窗口大小
//   - num_predict: 最大生成 token 数
//   - repeat_penalty: 重复惩罚
//   - stop: 停止序列
func WithOptions(options map[string]any) llm.RequestOption {
	return llm.WithExtraField(extOptions, options)
}

// WithThink 启用推理模式
// 用于支持推理的 Ollama 模型
func WithThink(enabled bool) llm.RequestOption {
	return llm.WithExtraField(extThink, enabled)
}
