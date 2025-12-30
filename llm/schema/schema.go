package schema

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"time"
)

type Role string

const (
	RoleSystem    Role = "system"
	RoleUser      Role = "user"
	RoleAssistant Role = "assistant"
	RoleTool      Role = "tool"
)

type Message struct {
	Role Role `json:"role"`

	// Content holds structured content (e.g. text + images).
	//
	// For simple text messages, use a single TextContent part (via schema.TextPart).
	Content []ContentPart `json:"content"`

	// Optional fields. Not all providers use/accept these.
	Name       string `json:"name,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`

	// Optional fields for providers that return separate reasoning content and
	// tool calls (e.g. DeepSeek).
	ReasoningContent string     `json:"reasoning_content,omitempty"`
	ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
}

func (m Message) MarshalJSON() ([]byte, error) {
	type outMsg struct {
		Role    Role `json:"role"`
		Content any  `json:"content"`

		Name       string `json:"name,omitempty"`
		ToolCallID string `json:"tool_call_id,omitempty"`

		ReasoningContent string     `json:"reasoning_content,omitempty"`
		ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	}

	var content any
	if len(m.Content) == 1 {
		if tp, ok := m.Content[0].(TextContent); ok {
			content = tp.Text
		}
	}
	if content == nil {
		parts := make([]map[string]any, 0, len(m.Content))
		for _, p := range m.Content {
			switch part := p.(type) {
			case TextContent:
				parts = append(parts, map[string]any{
					"type": "text",
					"text": part.Text,
				})
			case ImageURLContent:
				imageURL := map[string]any{"url": part.URL}
				if part.Detail != "" {
					imageURL["detail"] = part.Detail
				}
				parts = append(parts, map[string]any{
					"type":      "image_url",
					"image_url": imageURL,
				})
			case BinaryContent:
				if part.MIMEType == "" {
					return nil, fmt.Errorf("schema: binary part mime type required")
				}
				parts = append(parts, map[string]any{
					"type": "binary",
					"binary": map[string]any{
						"mime_type": part.MIMEType,
						"data":      base64.StdEncoding.EncodeToString(part.Data),
					},
				})
			default:
				return nil, fmt.Errorf("schema: unsupported content part type %T", p)
			}
		}
		content = parts
	}

	return json.Marshal(outMsg{
		Role:             m.Role,
		Content:          content,
		Name:             m.Name,
		ToolCallID:       m.ToolCallID,
		ReasoningContent: m.ReasoningContent,
		ToolCalls:        m.ToolCalls,
	})
}

func (m *Message) UnmarshalJSON(data []byte) error {
	type inMsg struct {
		Role    Role `json:"role"`
		Content any  `json:"content"`

		Name       string `json:"name,omitempty"`
		ToolCallID string `json:"tool_call_id,omitempty"`

		ReasoningContent string     `json:"reasoning_content,omitempty"`
		ToolCalls        []ToolCall `json:"tool_calls,omitempty"`
	}

	var in inMsg
	if err := json.Unmarshal(data, &in); err != nil {
		return err
	}

	out := Message{
		Role:             in.Role,
		Name:             in.Name,
		ToolCallID:       in.ToolCallID,
		ReasoningContent: in.ReasoningContent,
		ToolCalls:        in.ToolCalls,
	}

	switch v := in.Content.(type) {
	case string:
		out.Content = []ContentPart{TextContent{Text: v}}
	case []any:
		parts := make([]ContentPart, 0, len(v))
		for _, item := range v {
			b, err := json.Marshal(item)
			if err != nil {
				continue
			}
			var typ struct {
				Type string `json:"type"`
			}
			if err := json.Unmarshal(b, &typ); err != nil {
				continue
			}
			switch typ.Type {
			case "text", "":
				var p struct {
					Text string `json:"text"`
				}
				if err := json.Unmarshal(b, &p); err != nil {
					return err
				}
				parts = append(parts, TextContent{Text: p.Text})
			case "image_url":
				var p struct {
					ImageURL struct {
						URL    string `json:"url"`
						Detail string `json:"detail,omitempty"`
					} `json:"image_url"`
				}
				if err := json.Unmarshal(b, &p); err != nil {
					return err
				}
				parts = append(parts, ImageURLContent{URL: p.ImageURL.URL, Detail: p.ImageURL.Detail})
			case "binary":
				var p struct {
					Binary struct {
						MIMEType string `json:"mime_type"`
						Data     string `json:"data"`
					} `json:"binary"`
				}
				if err := json.Unmarshal(b, &p); err != nil {
					return err
				}
				decoded, err := base64.StdEncoding.DecodeString(p.Binary.Data)
				if err != nil {
					return fmt.Errorf("schema: decode binary data: %w", err)
				}
				parts = append(parts, BinaryContent{MIMEType: p.Binary.MIMEType, Data: decoded})
			default:
				return fmt.Errorf("schema: unknown content part type %q", typ.Type)
			}
		}
		out.Content = parts
	default:
		out.Content = nil
	}

	*m = out
	return nil
}

type ContentPart interface {
	isPart()
}

type TextContent struct {
	Text string
}

func (TextContent) isPart() {}

type ImageURLContent struct {
	URL    string
	Detail string
}

func (ImageURLContent) isPart() {}

type BinaryContent struct {
	MIMEType string
	Data     []byte
}

func (BinaryContent) isPart() {}

// Text returns the concatenated plain text of all text parts.
func (m Message) Text() string {
	var b []byte
	for _, p := range m.Content {
		if tp, ok := p.(TextContent); ok && tp.Text != "" {
			b = append(b, tp.Text...)
		}
	}
	return string(b)
}

type ToolCallType string

const (
	ToolCallTypeFunction ToolCallType = "function"
)

type ToolCall struct {
	ID       string       `json:"id"`
	Type     ToolCallType `json:"type"`
	Function ToolFunction `json:"function"`
}

type ToolFunction struct {
	Name      string `json:"name"`
	Arguments string `json:"arguments"`
}

type ToolType string

const (
	ToolTypeFunction ToolType = "function"
)

type Tool struct {
	Type     ToolType           `json:"type"`
	Function FunctionDefinition `json:"function"`
}

type FunctionDefinition struct {
	Name        string          `json:"name"`
	Description string          `json:"description,omitempty"`
	Parameters  json.RawMessage `json:"parameters,omitempty"`

	// Strict is a provider-dependent flag used for structured output guarantees.
	Strict bool `json:"strict,omitempty"`
}

type ToolChoiceMode string

const (
	ToolChoiceNone ToolChoiceMode = "none"
	ToolChoiceAuto ToolChoiceMode = "auto"
)

type ToolChoice struct {
	Mode         ToolChoiceMode `json:"mode"`
	FunctionName string         `json:"function_name,omitempty"`
}

type ResponseFormat struct {
	Type string `json:"type"`

	// JSONSchema is provider-specific JSON schema configuration when Type is
	// "json_schema".
	JSONSchema json.RawMessage `json:"json_schema,omitempty"`
}

type FinishReason string

const (
	FinishReasonStop          FinishReason = "stop"
	FinishReasonLength        FinishReason = "length"
	FinishReasonToolCalls     FinishReason = "tool_calls"
	FinishReasonContentFilter FinishReason = "content_filter"
)

type Usage struct {
	PromptTokens     int `json:"prompt_tokens"`
	CompletionTokens int `json:"completion_tokens"`
	TotalTokens      int `json:"total_tokens"`

	PromptCacheHitTokens  int `json:"prompt_cache_hit_tokens,omitempty"`
	PromptCacheMissTokens int `json:"prompt_cache_miss_tokens,omitempty"`

	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"`
}

type CompletionTokensDetails struct {
	ReasoningTokens int `json:"reasoning_tokens,omitempty"`
}

type Choice struct {
	Index        int          `json:"index"`
	Message      Message      `json:"message"`
	FinishReason FinishReason `json:"finish_reason"`
}

type ChatResponse struct {
	ID        string    `json:"id"`
	Model     string    `json:"model"`
	CreatedAt time.Time `json:"created_at"`

	Choices []Choice `json:"choices"`
	Usage   Usage    `json:"usage"`

	// Raw holds the provider-native payload for debugging/forward-compat.
	Raw json.RawMessage `json:"raw,omitempty"`
}

type StreamEventType string

const (
	StreamEventDelta StreamEventType = "delta"
	StreamEventDone  StreamEventType = "done"
)

type StreamEvent struct {
	Type        StreamEventType `json:"type"`
	ChoiceIndex int             `json:"choice_index,omitempty"`

	Delta        string        `json:"delta,omitempty"`
	Reasoning    string        `json:"reasoning,omitempty"`
	ToolCalls    []ToolCall    `json:"tool_calls,omitempty"`
	FinishReason *FinishReason `json:"finish_reason,omitempty"`
	Usage        *Usage        `json:"usage,omitempty"`

	Raw json.RawMessage `json:"raw,omitempty"`
}
