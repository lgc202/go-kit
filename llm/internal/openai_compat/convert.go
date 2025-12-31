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

func toWireTools(tools []schema.Tool) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		if t.Type != schema.ToolTypeFunction {
			continue
		}
		fn := map[string]any{
			"name": t.Function.Name,
		}
		if t.Function.Description != "" {
			fn["description"] = t.Function.Description
		}
		if len(t.Function.Parameters) > 0 {
			if !json.Valid(t.Function.Parameters) {
				return nil, fmt.Errorf("openai_compat: invalid tool parameters JSON for %q", t.Function.Name)
			}
			fn["parameters"] = json.RawMessage(t.Function.Parameters)
		}
		if t.Function.Strict {
			fn["strict"] = true
		}
		out = append(out, map[string]any{
			"type":     "function",
			"function": fn,
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
			return map[string]any{
				"type": "function",
				"function": map[string]any{
					"name": tc.FunctionName,
				},
			}
		}
		return "auto"
	}
}

func toWireResponseFormat(rf schema.ResponseFormat) (map[string]any, error) {
	out := map[string]any{
		"type": rf.Type,
	}
	if len(rf.JSONSchema) > 0 {
		if !json.Valid(rf.JSONSchema) {
			return nil, fmt.Errorf("openai_compat: invalid response_format.json_schema JSON")
		}
		out["json_schema"] = json.RawMessage(rf.JSONSchema)
	}
	return out, nil
}

func toWireMessage(provider string, m schema.Message) (map[string]any, error) {
	out := map[string]any{
		"role": string(m.Role),
	}
	if m.Name != "" {
		out["name"] = m.Name
	}
	if m.ToolCallID != "" {
		out["tool_call_id"] = m.ToolCallID
	}

	if len(m.Content) > 0 {
		if len(m.Content) == 1 {
			if tp, ok := m.Content[0].(schema.TextContent); ok {
				out["content"] = tp.Text
				return out, nil
			}
		}

		parts := make([]map[string]any, 0, len(m.Content))
		for _, p := range m.Content {
			switch part := p.(type) {
			case schema.TextContent:
				parts = append(parts, map[string]any{
					"type": "text",
					"text": part.Text,
				})
			case schema.ImageURLContent:
				imageURL := map[string]any{
					"url": part.URL,
				}
				if strings.TrimSpace(part.Detail) != "" {
					imageURL["detail"] = part.Detail
				}
				parts = append(parts, map[string]any{
					"type":      "image_url",
					"image_url": imageURL,
				})
			case schema.BinaryContent:
				if strings.TrimSpace(part.MIMEType) == "" {
					return nil, fmt.Errorf("%s: binary mime type required", provider)
				}
				if len(part.Data) == 0 {
					return nil, fmt.Errorf("%s: binary data required", provider)
				}
				dataURL := "data:" + part.MIMEType + ";base64," + base64.StdEncoding.EncodeToString(part.Data)
				parts = append(parts, map[string]any{
					"type": "image_url",
					"image_url": map[string]any{
						"url": dataURL,
					},
				})
			default:
				return nil, &llm.UnsupportedOptionError{
					Provider: llm.Provider(provider),
					Option:   "message.content",
					Reason:   fmt.Sprintf("unsupported content part type %T", p),
				}
			}
		}
		out["content"] = parts
		return out, nil
	}

	out["content"] = ""
	return out, nil
}
