package llm

import (
	"encoding/json"
	"net/http"
	"sort"
	"strings"
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

type ContentPartType string

const (
	ContentPartText      ContentPartType = "text"
	ContentPartReasoning ContentPartType = "reasoning"
	ContentPartJSON      ContentPartType = "json"
	ContentPartBinary    ContentPartType = "binary"
)

// ContentPart is a provider-agnostic "message content segment".
//
// Many providers represent message content as an array of parts (text, image, etc.).
// Keeping this as a first-class concept makes it easier to map to/from different APIs.
type ContentPart struct {
	Type ContentPartType `json:"type"`

	// Text is used by ContentPartText and ContentPartReasoning.
	Text string `json:"text,omitempty"`

	// JSON is an optional structured payload (provider-specific).
	JSON json.RawMessage `json:"json,omitempty"`

	// Data/MIME are for binary payloads (e.g. images/audio), if a provider supports them.
	Data []byte `json:"data,omitempty"`
	MIME string `json:"mime,omitempty"`
}

func TextPart(text string) ContentPart { return ContentPart{Type: ContentPartText, Text: text} }
func ReasoningPart(text string) ContentPart {
	return ContentPart{Type: ContentPartReasoning, Text: text}
}
func JSONPart(raw json.RawMessage) ContentPart {
	return ContentPart{Type: ContentPartJSON, JSON: append([]byte(nil), raw...)}
}

// Message is a canonical chat message.
//
// For tool results, use RoleTool with ToolCallID set.
// For assistant tool calls, use ToolCalls.
type Message struct {
	Role Role

	// Name is an optional sender name supported by some providers.
	Name string

	Parts []ContentPart

	ToolCallID string
	ToolCalls  []ToolCall
}

func System(text string) Message {
	return Message{Role: RoleSystem, Parts: []ContentPart{TextPart(text)}}
}
func User(text string) Message { return Message{Role: RoleUser, Parts: []ContentPart{TextPart(text)}} }
func Assistant(text string) Message {
	return Message{Role: RoleAssistant, Parts: []ContentPart{TextPart(text)}}
}
func ToolResult(toolCallID string, text string) Message {
	return Message{Role: RoleTool, ToolCallID: toolCallID, Parts: []ContentPart{TextPart(text)}}
}

func (m Message) Clone() Message {
	out := m
	if m.Parts != nil {
		out.Parts = make([]ContentPart, len(m.Parts))
		copy(out.Parts, m.Parts)
		for i := range out.Parts {
			out.Parts[i].JSON = append([]byte(nil), out.Parts[i].JSON...)
			out.Parts[i].Data = append([]byte(nil), out.Parts[i].Data...)
		}
	}
	if m.ToolCalls != nil {
		out.ToolCalls = make([]ToolCall, len(m.ToolCalls))
		copy(out.ToolCalls, m.ToolCalls)
		for i := range out.ToolCalls {
			out.ToolCalls[i].Arguments = append([]byte(nil), out.ToolCalls[i].Arguments...)
		}
	}
	return out
}

func (m Message) Text() string {
	var b strings.Builder
	for _, p := range m.Parts {
		if p.Type == ContentPartText {
			b.WriteString(p.Text)
		}
	}
	return b.String()
}

func (m Message) Reasoning() string {
	var b strings.Builder
	for _, p := range m.Parts {
		if p.Type == ContentPartReasoning {
			b.WriteString(p.Text)
		}
	}
	return b.String()
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

	// Details contains optional provider-specific usage breakdown fields.
	// Providers should populate this only when they have meaningful extra data.
	Details *UsageDetails
}

type UsageDetails struct {
	// PromptCacheHitTokens/PromptCacheMissTokens are emitted by some providers (e.g. DeepSeek)
	// to describe prompt caching behavior.
	PromptCacheHitTokens  int
	PromptCacheMissTokens int

	// ReasoningTokens is typically nested under completion token details for some providers.
	ReasoningTokens int
}

type TransportOptions struct {
	// Headers contains per-request header overrides/additions.
	//
	// This is an escape hatch for providers that require request-scoped headers
	// (e.g. vendor routing, beta flags). Providers may ignore unsupported headers.
	Headers http.Header
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

	Transport *TransportOptions

	// Extra carries provider-specific JSON fields. Keys should be top-level fields.
	// Values should be JSON-marshalable.
	Extra map[string]any
}

func (r ChatRequest) Clone() ChatRequest {
	out := r
	out.Messages = append([]Message(nil), r.Messages...)
	for i := range out.Messages {
		out.Messages[i] = out.Messages[i].Clone()
	}
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
	if r.Transport != nil {
		v := *r.Transport
		v.Headers = r.Transport.Headers.Clone()
		out.Transport = &v
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
	return r.Choices[0].Message.Text()
}

func (r ChatResponse) ChoiceIndexes() []int {
	if len(r.Choices) == 0 {
		return nil
	}
	idxs := make([]int, 0, len(r.Choices))
	for _, c := range r.Choices {
		idxs = append(idxs, c.Index)
	}
	sort.Ints(idxs)
	return idxs
}
