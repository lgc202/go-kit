package schema

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`

	PromptCacheHitTokens  int `json:"prompt_cache_hit_tokens,omitempty"`  // DeepSeek
	PromptCacheMissTokens int `json:"prompt_cache_miss_tokens,omitempty"` // DeepSeek

	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"` // OpenAI
}

type CompletionTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
}
