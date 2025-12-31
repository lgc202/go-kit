package openai_compat

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/schema"
)

func toSchemaUsage(u usage) schema.Usage {
	out := schema.Usage{
		PromptTokens:          u.PromptTokens,
		CompletionTokens:      u.CompletionTokens,
		TotalTokens:           u.TotalTokens,
		PromptCacheHitTokens:  u.PromptCacheHitTokens,
		PromptCacheMissTokens: u.PromptCacheMissTokens,
		CachedTokens:          u.CachedTokens,
	}
	if u.CompletionTokensDetails != nil && u.CompletionTokensDetails.ReasoningTokens != 0 {
		out.CompletionTokensDetails = &schema.CompletionTokensDetails{
			ReasoningTokens: u.CompletionTokensDetails.ReasoningTokens,
		}
	}
	return out
}

func toSchemaUsagePtr(u *usage) *schema.Usage {
	if u == nil {
		return nil
	}
	out := toSchemaUsage(*u)
	return &out
}

func toSchemaToolCalls(in []wireToolCall) []schema.ToolCall {
	if len(in) == 0 {
		return nil
	}
	out := make([]schema.ToolCall, len(in))
	for i, tc := range in {
		out[i] = schema.ToolCall{
			ID:   tc.ID,
			Type: schema.ToolCallType(tc.Type),
			Function: schema.ToolFunction{
				Name:      tc.Function.Name,
				Arguments: tc.Function.Arguments,
			},
		}
	}
	return out
}

func toSchemaMessage(m wireMessage) schema.Message {
	parts := toSchemaContentParts(m.Content)
	out := schema.Message{
		Role:             schema.Role(m.Role),
		Content:          parts,
		Name:             m.Name,
		ToolCallID:       m.ToolCallID,
		ReasoningContent: m.ReasoningContent,
		ToolCalls:        toSchemaToolCalls(m.ToolCalls),
	}
	return out
}

func toSchemaChatResponse(in chatCompletionResponse) schema.ChatResponse {
	out := schema.ChatResponse{
		ID:          in.ID,
		Model:       in.Model,
		Usage:       toSchemaUsage(in.Usage),
		ServiceTier: in.ServiceTier,
	}
	if in.Created != 0 {
		out.CreatedAt = time.Unix(in.Created, 0)
	}

	out.Choices = make([]schema.Choice, 0, len(in.Choices))
	for _, c0 := range in.Choices {
		out.Choices = append(out.Choices, schema.Choice{
			Index:        c0.Index,
			Message:      toSchemaMessage(c0.Message),
			FinishReason: schema.FinishReason(c0.FinishReason),
		})
	}
	return out
}

func toSchemaContentParts(in any) []schema.ContentPart {
	switch v := in.(type) {
	case string:
		if v == "" {
			return nil
		}
		return []schema.ContentPart{schema.TextContent{Text: v}}
	case []any:
		parts := make([]schema.ContentPart, 0, len(v))
		for _, p := range v {
			mp, ok := p.(map[string]any)
			if !ok {
				continue
			}
			tp, _ := mp["type"].(string)
			switch tp {
			case "text":
				text, _ := mp["text"].(string)
				parts = append(parts, schema.TextContent{Text: text})
			case "image_url":
				imageURL, ok := mp["image_url"].(map[string]any)
				if !ok {
					continue
				}
				url, _ := imageURL["url"].(string)
				detail, _ := imageURL["detail"].(string)
				parts = append(parts, schema.ImageURLContent{URL: url, Detail: detail})
			}
		}
		return parts
	default:
		return nil
	}
}

func toWireTools(tools []schema.Tool) ([]wireTool, error) {
	out := make([]wireTool, 0, len(tools))
	for _, t := range tools {
		if t.Type != schema.ToolTypeFunction {
			continue
		}
		var params json.RawMessage
		if len(t.Function.Parameters) > 0 {
			if !json.Valid(t.Function.Parameters) {
				return nil, fmt.Errorf("openai_compat: invalid tool parameters JSON for %q", t.Function.Name)
			}
			params = json.RawMessage(t.Function.Parameters)
		}
		out = append(out, wireTool{
			Type: "function",
			Function: wireFunction{
				Name:        t.Function.Name,
				Description: t.Function.Description,
				Parameters:  params,
				Strict:      t.Function.Strict,
			},
		})
	}
	return out, nil
}

func toWireToolChoice(tc schema.ToolChoice) any {
	switch tc.Mode {
	case schema.ToolChoiceNone:
		return "none"
	case schema.ToolChoiceAuto:
		return "auto"
	default:
		if tc.FunctionName != "" {
			var out wireToolChoiceFunction
			out.Type = "function"
			out.Function.Name = tc.FunctionName
			return out
		}
		return "auto"
	}
}

func toWireResponseFormat(rf schema.ResponseFormat) (*wireResponseFormat, error) {
	out := &wireResponseFormat{Type: rf.Type}
	if len(rf.JSONSchema) > 0 {
		if !json.Valid(rf.JSONSchema) {
			return nil, fmt.Errorf("openai_compat: invalid response_format.json_schema JSON")
		}
		out.JSONSchema = json.RawMessage(rf.JSONSchema)
	}
	return out, nil
}

func toWireMessage(provider string, m schema.Message) (wireRequestMessage, error) {
	out := wireRequestMessage{Role: string(m.Role)}
	if m.Name != "" {
		out.Name = m.Name
	}
	if m.ToolCallID != "" {
		out.ToolCallID = m.ToolCallID
	}

	if len(m.Content) > 0 {
		if len(m.Content) == 1 {
			if tp, ok := m.Content[0].(schema.TextContent); ok {
				out.Content = tp.Text
				return out, nil
			}
		}

		parts := make([]wireRequestContentPart, 0, len(m.Content))
		for _, p := range m.Content {
			switch part := p.(type) {
			case schema.TextContent:
				parts = append(parts, wireRequestContentPart{
					Type: "text",
					Text: part.Text,
				})
			case schema.ImageURLContent:
				parts = append(parts, wireRequestContentPart{
					Type: "image_url",
					ImageURL: &wireRequestImageURL{
						URL:    part.URL,
						Detail: strings.TrimSpace(part.Detail),
					},
				})
			case schema.BinaryContent:
				if strings.TrimSpace(part.MIMEType) == "" {
					return wireRequestMessage{}, fmt.Errorf("%s: binary mime type required", provider)
				}
				if len(part.Data) == 0 {
					return wireRequestMessage{}, fmt.Errorf("%s: binary data required", provider)
				}
				dataURL := "data:" + part.MIMEType + ";base64," + base64.StdEncoding.EncodeToString(part.Data)
				parts = append(parts, wireRequestContentPart{
					Type: "image_url",
					ImageURL: &wireRequestImageURL{
						URL: dataURL,
					},
				})
			default:
				return wireRequestMessage{}, &llm.UnsupportedOptionError{
					Provider: llm.Provider(provider),
					Option:   "message.content",
					Reason:   fmt.Sprintf("unsupported content part type %T", p),
				}
			}
		}
		out.Content = parts
		return out, nil
	}

	out.Content = ""
	return out, nil
}
