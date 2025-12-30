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

	// Content 包含结构化内容（如文本 + 图片）
	//
	// 对于简单文本消息，使用单个 TextContent 部分（通过 schema.TextPart）
	Content []ContentPart `json:"content"`

	// 可选字段，并非所有 provider 都支持/接受这些字段
	Name       string `json:"name,omitempty"`
	ToolCallID string `json:"tool_call_id,omitempty"`

	// 可选字段，用于返回独立推理内容和工具调用的 provider（如 DeepSeek）
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
	// 处理空内容情况
	if len(m.Content) == 0 {
		content = ""
	} else if len(m.Content) == 1 {
		// 单个内容部分：如果是文本，简化为字符串
		if tp, ok := m.Content[0].(TextContent); ok {
			content = tp.Text
		} else {
			// 非文本类型（图片/二进制），使用数组格式
			parts := make([]map[string]any, 0, 1)
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
	} else {
		// 多个内容部分，使用数组格式
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

	// Strict 是 provider 依赖的标志，用于结构化输出保证
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

	// JSONSchema 是 provider 特定的 JSON schema 配置，当 Type 为 "json_schema" 时使用
	JSONSchema json.RawMessage `json:"json_schema,omitempty"`
}

// StreamOptions 配置流式响应的行为。
type StreamOptions struct {
	// IncludeUsage 表示是否在流式响应的最后一个块中包含使用统计信息。
	// 设置为 true 时，流式响应结束时会包含一个包含 usage 字段的块。
	IncludeUsage bool `json:"include_usage,omitempty"`
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

	PromptCacheHitTokens  int `json:"prompt_cache_hit_tokens,omitempty"`  // DeepSeek
	PromptCacheMissTokens int `json:"prompt_cache_miss_tokens,omitempty"` // DeepSeek
	CachedTokens          int `json:"cached_tokens,omitempty"`            // Kimi

	CompletionTokensDetails *CompletionTokensDetails `json:"completion_tokens_details,omitempty"` // OpenAI
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

	// ServiceTier 返回实际使用的服务层级
	ServiceTier *string `json:"service_tier,omitempty"`

	// Raw 保留 provider 原生载荷，用于调试/向前兼容
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
