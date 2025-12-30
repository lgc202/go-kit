package openai_compat

// OpenAI 兼容的聊天完成接口类型定义
type errorResponse struct {
	Error struct {
		Message string `json:"message"`
		Type    string `json:"type"`
		Param   any    `json:"param"`
		Code    string `json:"code"`
	} `json:"error"`
}

type wireToolCall struct {
	ID       string `json:"id"`
	Type     string `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type wireMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`

	Name string `json:"name,omitempty"`

	ToolCallID string `json:"tool_call_id,omitempty"`

	ReasoningContent string         `json:"reasoning_content,omitempty"`
	ToolCalls        []wireToolCall `json:"tool_calls,omitempty"`
}

type usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`

	PromptCacheHitTokens  int `json:"prompt_cache_hit_tokens,omitempty"`
	PromptCacheMissTokens int `json:"prompt_cache_miss_tokens,omitempty"`

	CompletionTokensDetails *struct {
		ReasoningTokens int `json:"reasoning_tokens,omitempty"`
	} `json:"completion_tokens_details,omitempty"`
}

type chatCompletionResponse struct {
	ID      string `json:"id"`
	Created int64  `json:"created"`
	Model   string `json:"model"`

	Choices []struct {
		Index        int         `json:"index"`
		FinishReason string      `json:"finish_reason"`
		Message      wireMessage `json:"message"`
	} `json:"choices"`

	Usage       usage   `json:"usage"`
	ServiceTier *string `json:"service_tier,omitempty"`
}

type chatCompletionChunk struct {
	Choices []struct {
		Index int       `json:"index"`
		Delta wireDelta `json:"delta"`

		FinishReason *string `json:"finish_reason"`
	} `json:"choices"`

	Usage *usage `json:"usage,omitempty"`
}

type wireDelta struct {
	Role    string `json:"role,omitempty"`
	Content string `json:"content,omitempty"`

	ReasoningContent string `json:"reasoning_content,omitempty"`

	ToolCalls []wireToolCall `json:"tool_calls,omitempty"`
}
