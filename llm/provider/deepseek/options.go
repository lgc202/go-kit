package deepseek

import (
	"encoding/json"

	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/schema"
)

const (
	extThinking         = "deepseek.thinking"
	extResponseFormat   = "deepseek.response_format"
	extFrequencyPenalty = "deepseek.frequency_penalty"
	extPresencePenalty  = "deepseek.presence_penalty"
	extToolChoice       = "deepseek.tool_choice"
	extTools            = "deepseek.tools"
	extLogprobs         = "deepseek.logprobs"
	extTopLogprobs      = "deepseek.top_logprobs"
	extStreamOptions    = "deepseek.stream_options"
)

type ThinkingType string

const (
	ThinkingTypeDisabled ThinkingType = "disabled"
)

type Thinking struct {
	Type ThinkingType `json:"type"`
}

func WithThinkingDisabled() llm.RequestOption {
	return llm.WithExtension(extThinking, Thinking{Type: ThinkingTypeDisabled})
}

type ResponseFormatType string

const (
	ResponseFormatTypeText ResponseFormatType = "text"
)

type ResponseFormat struct {
	Type ResponseFormatType `json:"type"`
}

func WithResponseFormatText() llm.RequestOption {
	return llm.WithResponseFormat(schema.ResponseFormat{Type: string(ResponseFormatTypeText)})
}

func WithFrequencyPenalty(v float64) llm.RequestOption {
	return llm.WithFrequencyPenalty(v)
}

func WithPresencePenalty(v float64) llm.RequestOption {
	return llm.WithPresencePenalty(v)
}

func WithTools(tools any) llm.RequestOption {
	return llm.WithExtension(extTools, tools)
}

func WithToolChoice(choice any) llm.RequestOption {
	return llm.WithExtension(extToolChoice, choice)
}

func WithLogprobs(enabled bool) llm.RequestOption {
	return llm.WithLogprobs(enabled)
}

func WithTopLogprobs(v int) llm.RequestOption {
	return llm.WithTopLogprobs(v)
}

func WithStreamOptions(streamOptions any) llm.RequestOption {
	return llm.WithExtension(extStreamOptions, streamOptions)
}

func WithToolsCompat(tools ...schema.Tool) llm.RequestOption {
	return llm.WithTools(tools...)
}

func WithToolChoiceNone() llm.RequestOption {
	return llm.WithToolChoice(schema.ToolChoice{Mode: schema.ToolChoiceNone})
}

func WithToolChoiceAuto() llm.RequestOption {
	return llm.WithToolChoice(schema.ToolChoice{Mode: schema.ToolChoiceAuto})
}

func WithToolChoiceFunction(name string) llm.RequestOption {
	return llm.WithToolChoice(schema.ToolChoice{FunctionName: name})
}

func WithResponseFormatJSONSchema(s json.RawMessage) llm.RequestOption {
	return llm.WithResponseFormat(schema.ResponseFormat{
		Type:       "json_schema",
		JSONSchema: s,
	})
}
