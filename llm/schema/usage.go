package schema

// Usage 表示 token 使用统计
type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`

	PromptCacheHitTokens  int `json:"prompt_cache_hit_tokens,omitempty"`  // DeepSeek 缓存命中
	PromptCacheMissTokens int `json:"prompt_cache_miss_tokens,omitempty"` // DeepSeek 缓存未命中

	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"` // OpenAI 细分统计
}

// CompletionTokensDetails 表示完成 token 的细分统计
type CompletionTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens,omitempty"` // 推理消耗的 token
}
