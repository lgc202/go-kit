package llm

import (
	"encoding/json"
	"time"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type FinishReason string

const (
	FinishReasonStop      FinishReason = "stop"
	FinishReasonLength    FinishReason = "length"
	FinishReasonToolCalls FinishReason = "tool_calls"
	FinishReasonUnknown   FinishReason = "unknown"
)

// Message is a canonical chat message.
//
// For tool results, use RoleTool with ToolCallID set.
// For assistant tool calls, use ToolCalls.
type Message struct {
	Role Role

	// Content is text (and for RoleTool: tool output).
	Content string

	// Name is an optional sender name supported by some providers.
	Name string

	// Reasoning is provider-specific "thinking"/"reasoning" content.
	//
	// Some providers (e.g. DeepSeek) return it as `reasoning_content` or `thinking`.
	// It may be empty even when Content is present.
	Reasoning string

	ToolCallID string
	ToolCalls  []ToolCall
}

type ToolDefinition struct {
	Name        string
	Description string

	// InputSchema is typically a JSON Schema object.
	InputSchema json.RawMessage
}

type ToolChoiceMode string

const (
	ToolChoiceAuto     ToolChoiceMode = "auto"
	ToolChoiceNone     ToolChoiceMode = "none"
	ToolChoiceRequired ToolChoiceMode = "required"
	ToolChoiceFunction ToolChoiceMode = "function"
)

// ToolChoice models the user's preference for tool usage.
//
// For ToolChoiceFunction, set FunctionName.
type ToolChoice struct {
	Mode         ToolChoiceMode
	FunctionName string
}

func AutoToolChoice() ToolChoice     { return ToolChoice{Mode: ToolChoiceAuto} }
func NoneToolChoice() ToolChoice     { return ToolChoice{Mode: ToolChoiceNone} }
func RequiredToolChoice() ToolChoice { return ToolChoice{Mode: ToolChoiceRequired} }
func FunctionToolChoice(name string) ToolChoice {
	return ToolChoice{Mode: ToolChoiceFunction, FunctionName: name}
}

// ToolCall is a canonical representation of a tool/function call.
//
// Some providers stream ArgumentsText in chunks and may not guarantee valid JSON.
// When possible, providers should fill Arguments (valid JSON bytes).
type ToolCall struct {
	ID            string
	Name          string
	Arguments     json.RawMessage
	ArgumentsText string
}

type Usage struct {
	PromptTokens     int
	CompletionTokens int
	TotalTokens      int
}

type ChatRequest struct {
	Model    string
	Messages []Message

	Temperature *float64
	TopP        *float64
	MaxTokens   *int
	Seed        *int64

	PresencePenalty  *float64
	FrequencyPenalty *float64
	Stop             []string

	ResponseFormat *ResponseFormat

	LogProbs    *bool
	TopLogProbs *int

	StreamOptions *StreamOptions

	Tools      []ToolDefinition
	ToolChoice *ToolChoice

	// Extra carries provider-specific JSON fields. Keys should be top-level fields.
	// Values should be JSON-marshalable.
	Extra map[string]any
}

func (r ChatRequest) Clone() ChatRequest {
	out := r
	out.Messages = append([]Message(nil), r.Messages...)
	if r.Tools != nil {
		out.Tools = append([]ToolDefinition(nil), r.Tools...)
	}
	if r.ToolChoice != nil {
		v := *r.ToolChoice
		out.ToolChoice = &v
	}
	if r.Stop != nil {
		out.Stop = append([]string(nil), r.Stop...)
	}
	if r.ResponseFormat != nil {
		v := *r.ResponseFormat
		out.ResponseFormat = &v
	}
	if r.StreamOptions != nil {
		v := *r.StreamOptions
		out.StreamOptions = &v
	}
	if r.Extra != nil {
		out.Extra = make(map[string]any, len(r.Extra))
		for k, v := range r.Extra {
			out.Extra[k] = v
		}
	}
	return out
}

type ResponseFormatType string

const (
	ResponseFormatText       ResponseFormatType = "text"
	ResponseFormatJSONObject ResponseFormatType = "json_object"
	ResponseFormatJSONSchema ResponseFormatType = "json_schema"
)

// ResponseFormat models the OpenAI-compatible response_format object.
//
// Examples:
//
//	{"type":"text"}
//	{"type":"json_object"}
//	{"type":"json_schema","json_schema":{...}}
type ResponseFormat struct {
	Type ResponseFormatType `json:"type"`

	// JSONSchema is provider-specific and only meaningful when Type == json_schema.
	JSONSchema json.RawMessage `json:"json_schema,omitempty"`
}

type StreamOptions struct {
	IncludeUsage bool `json:"include_usage,omitempty"`
}

type ChatChoice struct {
	Index        int
	Message      Message
	FinishReason FinishReason
}

type ChatResponse struct {
	ID      string
	Model   string
	Created time.Time

	Choices []ChatChoice
	Usage   *Usage

	RawJSON json.RawMessage
	Meta    map[string]any
}

func (r ChatResponse) FirstText() string {
	if len(r.Choices) == 0 {
		return ""
	}
	return r.Choices[0].Message.Content
}
