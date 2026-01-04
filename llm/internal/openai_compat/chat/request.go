package chat

import (
	"encoding/json"
	"fmt"
)

type wireContentType string

const (
	wireContentTypeText     wireContentType = "text"
	wireContentTypeImageURL wireContentType = "image_url"
)

type wireToolType string

const (
	wireToolTypeFunction wireToolType = "function"
)

type wireToolChoiceMode string

const (
	wireToolChoiceNone wireToolChoiceMode = "none"
	wireToolChoiceAuto wireToolChoiceMode = "auto"
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
	ToolChoice        *wireToolChoice     `json:"tool_choice,omitempty"`
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
	Role    string             `json:"role"`
	Content wireRequestContent `json:"content"`

	Name       string `json:"name,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`

	// ToolCalls 用于回放包含 tool_calls 的 assistant 消息，
	// 以保证后续 role=tool 的消息有对应的前置 tool_calls。
	ToolCalls []wireToolCall `json:"tool_calls,omitempty"`
}

type wireRequestContent struct {
	text   string
	isText bool
	parts  []wireRequestContentPart
}

func wireRequestText(s string) wireRequestContent {
	return wireRequestContent{text: s, isText: true}
}

func wireRequestParts(parts []wireRequestContentPart) wireRequestContent {
	return wireRequestContent{parts: parts}
}

func (c wireRequestContent) MarshalJSON() ([]byte, error) {
	if c.isText {
		return json.Marshal(c.text)
	}
	return json.Marshal(c.parts)
}

type wireRequestContentPart struct {
	Type wireContentType `json:"type"`

	Text string `json:"text,omitempty"`

	ImageURL *wireRequestImageURL `json:"image_url,omitempty"`
}

type wireRequestImageURL struct {
	URL    string `json:"url"`
	Detail string `json:"detail,omitempty"`
}

type wireTool struct {
	Type     wireToolType `json:"type"`
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
	Type     wireToolType `json:"type"`
	Function struct {
		Name string `json:"name"`
	} `json:"function"`
}

type wireToolChoice struct {
	Mode         wireToolChoiceMode
	FunctionName string
}

func (tc wireToolChoice) MarshalJSON() ([]byte, error) {
	switch tc.Mode {
	case wireToolChoiceNone, wireToolChoiceAuto:
		return json.Marshal(tc.Mode)
	default:
		if tc.FunctionName == "" {
			return json.Marshal(wireToolChoiceAuto)
		}
		var out wireToolChoiceFunction
		out.Type = wireToolTypeFunction
		out.Function.Name = tc.FunctionName
		return json.Marshal(out)
	}
}
