package openai_compat

import (
	"encoding/json"
	"fmt"
)

type chatCompletionRequest struct {
	provider string `json:"-"`

	Model    string               `json:"model"`
	Messages []wireRequestMessage `json:"messages"`
	Stream   bool                 `json:"stream"`

	Temperature      *float64 `json:"temperature,omitempty"`
	TopP             *float64 `json:"top_p,omitempty"`
	FrequencyPenalty *float64 `json:"frequency_penalty,omitempty"`
	PresencePenalty  *float64 `json:"presence_penalty,omitempty"`
	Seed             *int     `json:"seed,omitempty"`

	MaxTokens           *int `json:"max_tokens,omitempty"`
	MaxCompletionTokens *int `json:"max_completion_tokens,omitempty"`

	Stop        []string `json:"stop,omitempty"`
	Logprobs    *bool    `json:"logprobs,omitempty"`
	TopLogprobs *int     `json:"top_logprobs,omitempty"`
	N           *int     `json:"n,omitempty"`

	Metadata  map[string]string `json:"metadata,omitempty"`
	LogitBias map[string]int    `json:"logit_bias,omitempty"`

	ServiceTier *string `json:"service_tier,omitempty"`
	User        *string `json:"user,omitempty"`

	Tools             []wireTool          `json:"tools,omitempty"`
	ToolChoice        any                 `json:"tool_choice,omitempty"`
	ParallelToolCalls *bool               `json:"parallel_tool_calls,omitempty"`
	ResponseFormat    *wireResponseFormat `json:"response_format,omitempty"`

	StreamOptions json.RawMessage `json:"stream_options,omitempty"`

	extra                   map[string]any `json:"-"`
	allowExtraFieldOverride bool           `json:"-"`
}

func (r chatCompletionRequest) MarshalJSON() ([]byte, error) {
	type alias chatCompletionRequest
	base, err := json.Marshal(alias(r))
	if err != nil {
		return nil, err
	}
	if len(r.extra) == 0 {
		return base, nil
	}

	var obj map[string]json.RawMessage
	if err := json.Unmarshal(base, &obj); err != nil {
		return nil, err
	}

	for k, v := range r.extra {
		if !r.allowExtraFieldOverride {
			if _, exists := obj[k]; exists {
				return nil, fmt.Errorf("%s: extra field %q conflicts with a built-in option (set llm.WithAllowExtraFieldOverride(true) to override)", r.provider, k)
			}
		}
		b, err := json.Marshal(v)
		if err != nil {
			return nil, err
		}
		obj[k] = b
	}

	return json.Marshal(obj)
}

type wireRequestMessage struct {
	Role    string `json:"role"`
	Content any    `json:"content"`

	Name       string `json:"name,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`
}

type wireRequestContentPart struct {
	Type string `json:"type"`

	Text string `json:"text,omitempty"`

	ImageURL *wireRequestImageURL `json:"image_url,omitempty"`
}

type wireRequestImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type wireTool struct {
	Type     string       `json:"type"`
	Function wireFunction `json:"function"`
}

type wireFunction struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`
	Strict      bool            `json:"strict,omitempty"`
}

type wireResponseFormat struct {
	Type string `json:"type"`

	JSONSchema json.RawMessage `json:"json_schema,omitempty"`
}

type wireToolChoiceFunction struct {
	Type     string `json:"type"`
	Function struct {
		Name string `json:"name"`
	} `json:"function"`
}
