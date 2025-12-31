package llm

import (
	"context"
	"maps"
	"net/http"
	"slices"
	"time"

	"github.com/lgc202/go-kit/llm/schema"
)

// RequestOption 是请求配置的可选参数函数类型
type RequestOption func(*RequestConfig)

// RequestConfig 表示单次 chat 请求的配置
type RequestConfig struct {
	// === 基础参数 ===

	// Model 指定要使用的模型 ID（如 "gpt-4o", "deepseek-chat"）
	Model string

	// Temperature 设置采样温度，控制输出的随机性
	// 范围 0-2，默认值 1。值越高输出越随机，值越低输出越确定
	Temperature *float64

	// TopP 设置核采样阈值，也称为 nucleus sampling
	// 范围 0-1，默认值 1。模型只考虑概率质量总和达到 topP 的最小 token 集合
	TopP *float64

	// MaxTokens 设置生成的最大 token 数（已弃用，建议使用 MaxCompletionTokens）
	// 注意：OpenAI 已弃用此参数，推荐使用 MaxCompletionTokens
	MaxTokens *int

	// MaxCompletionTokens 设置生成的最大 token 数上限
	// 这是 max_tokens 的推荐替代品，适用于所有模型
	MaxCompletionTokens *int

	// Stop 设置最多 4 个停止序列，遇到这些序列时停止生成
	Stop *[]string

	// === 惩罚参数 ===

	// FrequencyPenalty 设置频率惩罚，用于减少重复内容
	// 范围 -2.0 到 2.0，默认值 0。正值会根据 token 在文本中出现的频率进行惩罚
	FrequencyPenalty *float64

	// PresencePenalty 设置存在惩罚，用于鼓励话题多样性
	// 范围 -2.0 到 2.0，默认值 0。正值会惩罚已出现过的 token
	PresencePenalty *float64

	// === LogProbs 相关 ===

	// Logprobs 设置是否返回对数概率
	// 设置为 true 时，每个 token 会返回对数概率信息
	Logprobs *bool

	// TopLogprobs 设置每个 token 位置返回的最可能 token 数量
	// 范围 0-20，默认值 0。仅在 Logprobs 为 true 时有效
	TopLogprobs *int

	// === 工具调用相关 ===

	// Tools 设置模型可调用的工具列表
	Tools []schema.Tool

	// ToolChoice 设置工具调用模式
	ToolChoice *schema.ToolChoice

	// ParallelToolCalls 设置是否启用并行工具调用
	// 默认为 true，设置为 false 时模型会串行调用工具
	ParallelToolCalls *bool

	// === 输出格式 ===

	// ResponseFormat 设置响应的输出格式
	ResponseFormat *schema.ResponseFormat

	// === 候选结果 ===

	// N 设置返回的 chat completion 选择数量
	// 默认值为 1。设置大于 1 时会生成多个候选响应
	N *int

	// === 确定性采样 ===

	// Seed 设置采样种子，用于实现确定性输出
	// 相同的 seed 和参数设置会产生相同的输出，用于测试和调试
	Seed *int

	// === 元数据 ===

	// Metadata 设置请求的元数据，最多支持 16 个键值对
	// 用于企业级追踪和组织，键和值都是字符串，长度不超过 64 字符
	Metadata map[string]string

	// LogitBias 设置 token 偏置，用于修改特定 token 出现的概率
	// 接受一个 map，将 token ID（字符串）映射到 -100 到 100 之间的偏置值
	// -100 表示禁止，100 表示独占
	LogitBias map[string]int

	// ServiceTier 设置处理请求的服务层级
	// "auto": 使用项目设置中的服务层级（默认）
	// "default": 标准定价和性能
	// "flex" 或 "priority": 相应的服务层级
	ServiceTier *string

	// User 设置最终用户的唯一标识符
	// 用于监控和检测滥用，以及提升缓存命中率
	// 注意: OpenAI 推荐使用 PromptCacheKey 代替此字段
	User *string

	// === 流式选项 ===

	// StreamOptions 设置流式响应的选项
	StreamOptions *schema.StreamOptions

	// === 客户端配置（不发送到 API） ===

	// Timeout 设置请求的超时时间
	Timeout *time.Duration

	// Headers 设置发送到 API 的自定义 HTTP 头
	Headers http.Header

	// ExtraFields 允许 provider 特定的扩展，这些字段会直接合并到请求体中
	// Provider 也可以通过 adapter 应用自定义逻辑
	ExtraFields map[string]any

	// AllowExtraFieldOverride 控制 ExtraFields 是否允许覆盖已由标准选项设置的请求字段。
	// 默认为 false，避免“看似设置了 WithModel/WithMaxTokens 但被 ExtraFields 覆盖”的隐蔽问题。
	AllowExtraFieldOverride bool

	// KeepRaw 设置是否保留 provider 原始的 JSON 响应
	// 为 true 时，schema.ChatResponse.Raw 和 schema.StreamEvent.Raw 会包含原始响应
	// 默认为 false 以减少内存占用
	KeepRaw bool

	// StreamingFunc 是流式输出回调（客户端使用，不发送到 provider）
	// 在 llm.Wrap(...).Chat 中使用流式 API 时会被调用
	StreamingFunc func(ctx context.Context, chunk []byte) error

	// StreamingReasoningFunc 是推理内容的流式回调（客户端使用，不发送到 provider）
	// 用于分离推理内容和实际内容的流式输出
	StreamingReasoningFunc func(ctx context.Context, reasoningChunk, chunk []byte) error
}

// ApplyRequestOptions 将选项应用到一个新的 RequestConfig 上，返回配置结果。
//
// 采用“每次请求构建新配置”的方式（类似 langchaingo/eino），避免维护深拷贝逻辑。
func ApplyRequestOptions(opts ...RequestOption) RequestConfig {
	var cfg RequestConfig
	for _, opt := range opts {
		if opt == nil {
			continue
		}
		opt(&cfg)
	}
	return cfg
}

// === 基础参数 ===

// WithModel 设置要使用的模型 ID
func WithModel(model string) RequestOption {
	return func(c *RequestConfig) {
		c.Model = model
	}
}

// WithTemperature 设置采样温度（0-2）
func WithTemperature(v float64) RequestOption {
	return func(c *RequestConfig) {
		c.Temperature = &v
	}
}

// WithTopP 设置核采样阈值（0-1）
func WithTopP(v float64) RequestOption {
	return func(c *RequestConfig) {
		c.TopP = &v
	}
}

// WithMaxTokens 设置生成的最大 token 数（已弃用，建议使用 WithMaxCompletionTokens）
func WithMaxTokens(v int) RequestOption {
	return func(c *RequestConfig) {
		c.MaxTokens = &v
	}
}

// WithMaxCompletionTokens 设置生成的最大 token 数上限（推荐使用）
func WithMaxCompletionTokens(v int) RequestOption {
	return func(c *RequestConfig) {
		c.MaxCompletionTokens = &v
	}
}

// WithStop 设置停止序列
func WithStop(stop ...string) RequestOption {
	return func(c *RequestConfig) {
		cp := slices.Clone(stop)
		c.Stop = &cp
	}
}

// === 惩罚参数 ===

// WithFrequencyPenalty 设置频率惩罚（-2.0 到 2.0）
func WithFrequencyPenalty(v float64) RequestOption {
	return func(c *RequestConfig) {
		c.FrequencyPenalty = &v
	}
}

// WithPresencePenalty 设置存在惩罚（-2.0 到 2.0）
func WithPresencePenalty(v float64) RequestOption {
	return func(c *RequestConfig) {
		c.PresencePenalty = &v
	}
}

// === LogProbs 相关 ===

// WithLogprobs 设置是否返回对数概率
func WithLogprobs(enabled bool) RequestOption {
	return func(c *RequestConfig) {
		c.Logprobs = &enabled
	}
}

// WithTopLogprobs 设置每个 token 位置返回的最可能 token 数量（0-20）
func WithTopLogprobs(v int) RequestOption {
	return func(c *RequestConfig) {
		c.TopLogprobs = &v
	}
}

// === 工具调用相关 ===

// WithTools 设置模型可调用的工具列表
func WithTools(tools ...schema.Tool) RequestOption {
	return func(c *RequestConfig) {
		c.Tools = slices.Clone(tools)
	}
}

// WithToolChoice 设置工具调用模式
func WithToolChoice(choice schema.ToolChoice) RequestOption {
	return func(c *RequestConfig) {
		v := choice
		c.ToolChoice = &v
	}
}

// WithParallelToolCalls 设置是否启用并行工具调用
// enabled 为 true 时启用（默认），false 时禁用串行调用
func WithParallelToolCalls(enabled bool) RequestOption {
	return func(c *RequestConfig) {
		c.ParallelToolCalls = &enabled
	}
}

// === 输出格式 ===

// WithResponseFormat 设置响应的输出格式
func WithResponseFormat(format schema.ResponseFormat) RequestOption {
	return func(c *RequestConfig) {
		v := format
		c.ResponseFormat = &v
	}
}

// === 候选结果 ===

// WithN 设置返回的 chat completion 选择数量
// 默认值为 1，大于 1 时会生成多个候选响应
func WithN(n int) RequestOption {
	return func(c *RequestConfig) {
		c.N = &n
	}
}

// === 确定性采样 ===

// WithSeed 设置采样种子，用于实现确定性输出
// 相同的 seed 和参数设置会产生相同的输出，用于测试和调试
func WithSeed(seed int) RequestOption {
	return func(c *RequestConfig) {
		c.Seed = &seed
	}
}

// === 元数据 ===

// WithMetadata 设置请求的元数据
// 最多支持 16 个键值对，键和值都是字符串，长度不超过 64 字符
func WithMetadata(metadata map[string]string) RequestOption {
	metadata = maps.Clone(metadata)
	return func(c *RequestConfig) {
		if len(metadata) == 0 {
			return
		}
		if c.Metadata == nil {
			c.Metadata = make(map[string]string, len(metadata))
		}
		maps.Copy(c.Metadata, metadata)
	}
}

// WithLogitBias 设置 token 偏置
// 用于修改特定 token 出现的概率，值范围为 -100 到 100
func WithLogitBias(bias map[string]int) RequestOption {
	bias = maps.Clone(bias)
	return func(c *RequestConfig) {
		if len(bias) == 0 {
			return
		}
		if c.LogitBias == nil {
			c.LogitBias = make(map[string]int, len(bias))
		}
		maps.Copy(c.LogitBias, bias)
	}
}

// WithServiceTier 设置处理请求的服务层级
func WithServiceTier(tier string) RequestOption {
	return func(c *RequestConfig) {
		c.ServiceTier = &tier
	}
}

// WithUser 设置最终用户的唯一标识符
// 用于监控和检测滥用，以及提升缓存命中率
func WithUser(user string) RequestOption {
	return func(c *RequestConfig) {
		c.User = &user
	}
}

// === 流式选项 ===

// WithStreamOptions 设置流式响应的选项
func WithStreamOptions(opts schema.StreamOptions) RequestOption {
	return func(c *RequestConfig) {
		c.StreamOptions = &opts
	}
}

// WithStreamIncludeUsage 设置流式响应是否包含使用统计
// 设置为 true 后，流式响应的最后一个块会包含 usage 字段
func WithStreamIncludeUsage() RequestOption {
	return WithStreamOptions(schema.StreamOptions{IncludeUsage: true})
}

// === 客户端配置 ===

// WithTimeout 设置请求的超时时间
func WithTimeout(d time.Duration) RequestOption {
	return func(c *RequestConfig) {
		c.Timeout = &d
	}
}

// WithHeader 设置单个 HTTP 头
func WithHeader(key, value string) RequestOption {
	return func(c *RequestConfig) {
		if c.Headers == nil {
			c.Headers = make(http.Header)
		}
		c.Headers.Set(key, value)
	}
}

// WithExtraHeaders 批量设置 HTTP 头
func WithExtraHeaders(headers map[string]string) RequestOption {
	headers = maps.Clone(headers)
	return func(c *RequestConfig) {
		if len(headers) == 0 {
			return
		}
		if c.Headers == nil {
			c.Headers = make(http.Header)
		}
		for k, v := range headers {
			c.Headers.Set(k, v)
		}
	}
}

// WithExtraFields 批量设置扩展字段
// 这些字段会直接合并到请求体中，用于支持 provider 特定的功能
func WithExtraFields(fields map[string]any) RequestOption {
	fields = maps.Clone(fields)
	return func(c *RequestConfig) {
		if len(fields) == 0 {
			return
		}
		if c.ExtraFields == nil {
			c.ExtraFields = make(map[string]any, len(fields))
		}
		maps.Copy(c.ExtraFields, fields)
	}
}

// WithExtraField 设置单个扩展字段
// 对于多个字段，建议使用 WithExtraFields
func WithExtraField(key string, value any) RequestOption {
	return func(c *RequestConfig) {
		if c.ExtraFields == nil {
			c.ExtraFields = make(map[string]any)
		}
		c.ExtraFields[key] = value
	}
}

// WithAllowExtraFieldOverride 设置是否允许 ExtraFields 覆盖已存在的请求字段。
func WithAllowExtraFieldOverride(enabled bool) RequestOption {
	return func(c *RequestConfig) {
		c.AllowExtraFieldOverride = enabled
	}
}

// WithKeepRaw 设置是否保留 provider 原始的 JSON 响应
func WithKeepRaw(enabled bool) RequestOption {
	return func(c *RequestConfig) {
		c.KeepRaw = enabled
	}
}

// WithStreamingFunc 设置流式输出回调（客户端使用）
func WithStreamingFunc(f func(ctx context.Context, chunk []byte) error) RequestOption {
	return func(c *RequestConfig) {
		c.StreamingFunc = f
	}
}

// WithStreamingReasoningFunc 设置推理内容的流式回调（客户端使用）
func WithStreamingReasoningFunc(f func(ctx context.Context, reasoningChunk, chunk []byte) error) RequestOption {
	return func(c *RequestConfig) {
		c.StreamingReasoningFunc = f
	}
}
