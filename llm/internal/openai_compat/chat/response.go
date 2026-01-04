package chat

import "encoding/json"

type wireToolCall struct {
	ID       string       `json:"id"`
	Type     wireToolType `json:"type"`
	Function struct {
		Name      string `json:"name"`
		Arguments string `json:"arguments"`
	} `json:"function"`
}

type wireMessage struct {
	Role    string          `json:"role"`
	Content json.RawMessage `json:"content"`

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
	CachedTokens          int `json:"cached_tokens,omitempty"`

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

type wireContentPart struct {
	Type wireContentType `json:"type"`

	Text string `json:"text,omitempty"`

	ImageURL *struct {
		URL    string `json:"url"`
		Detail string `json:"detail,omitempty"`
	} `json:"image_url,omitempty"`
}
