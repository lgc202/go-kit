package chat

import (
	"encoding/base64"
	"encoding/json"
	"fmt"
	"strings"
	"time"

	"github.com/lgc202/go-kit/llm/schema"
)

func toSchemaUsage(u usage) schema.Usage {
	promptCacheHitTokens := u.PromptCacheHitTokens
	if promptCacheHitTokens == 0 && u.CachedTokens != 0 {
		promptCacheHitTokens = u.CachedTokens
	}
	out := schema.Usage{
		PromptTokens:          u.PromptTokens,
		CompletionTokens:      u.CompletionTokens,
		TotalTokens:           u.TotalTokens,
		PromptCacheHitTokens:  promptCacheHitTokens,
		PromptCacheMissTokens: u.PromptCacheMissTokens,
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

func toWireToolCalls(in []schema.ToolCall) []wireToolCall {
	if len(in) == 0 {
		return nil
	}
	out := make([]wireToolCall, len(in))
	for i, tc := range in {
		out[i] = wireToolCall{
			ID:   tc.ID,
			Type: wireToolType(tc.Type),
		}
		out[i].Function.Name = tc.Function.Name
		out[i].Function.Arguments = tc.Function.Arguments
	}
	return out
}

func toSchemaMessage(m wireMessage) schema.Message {
	parts := toSchemaContentParts(m.Content)

	// 优先使用 reasoning_content，为空则使用 reasoning
	// DeepSeek API: reasoning_content
	// Ollama DeepSeek R1: reasoning
	reasoningContent := m.ReasoningContent
	if reasoningContent == "" {
		reasoningContent = m.Reasoning
	}

	out := schema.Message{
		Role:             schema.Role(m.Role),
		Content:          parts,
		Name:             m.Name,
		ToolCallID:       m.ToolCallID,
		ReasoningContent: reasoningContent,
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

func toSchemaContentParts(raw json.RawMessage) []schema.ContentPart {
	if len(raw) == 0 || string(raw) == "null" {
		return nil
	}

	switch raw[0] {
	case '"':
		var s string
		if err := json.Unmarshal(raw, &s); err != nil {
			return nil
		}
		if s == "" {
			return nil
		}
		return []schema.ContentPart{schema.TextContent{Text: s}}
	case '[':
		var partsWire []wireContentPart
		if err := json.Unmarshal(raw, &partsWire); err != nil {
			return nil
		}
		parts := make([]schema.ContentPart, 0, len(partsWire))
		for _, p := range partsWire {
			switch p.Type {
			case wireContentTypeText, "":
				if p.Text != "" {
					parts = append(parts, schema.TextContent{Text: p.Text})
				}
			case wireContentTypeImageURL:
				if p.ImageURL != nil {
					parts = append(parts, schema.ImageURLContent{URL: p.ImageURL.URL, Detail: p.ImageURL.Detail})
				}
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
			Type: wireToolTypeFunction,
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

func toWireToolChoice(tc schema.ToolChoice) *wireToolChoice {
	switch tc.Mode {
	case schema.ToolChoiceNone:
		return &wireToolChoice{Mode: wireToolChoiceNone}
	case schema.ToolChoiceAuto:
		return &wireToolChoice{Mode: wireToolChoiceAuto}
	default:
		if tc.FunctionName != "" {
			return &wireToolChoice{FunctionName: tc.FunctionName}
		}
		return &wireToolChoice{Mode: wireToolChoiceAuto}
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
	if len(m.ToolCalls) > 0 {
		out.ToolCalls = toWireToolCalls(m.ToolCalls)
	}

	if len(m.Content) > 0 {
		if len(m.Content) == 1 {
			if tp, ok := m.Content[0].(schema.TextContent); ok {
				out.Content = wireRequestText(tp.Text)
				return out, nil
			}
		}

		parts := make([]wireRequestContentPart, 0, len(m.Content))
		for _, p := range m.Content {
			switch part := p.(type) {
			case schema.TextContent:
				parts = append(parts, wireRequestContentPart{
					Type: wireContentTypeText,
					Text: part.Text,
				})
			case schema.ImageURLContent:
				parts = append(parts, wireRequestContentPart{
					Type: wireContentTypeImageURL,
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
					Type: wireContentTypeImageURL,
					ImageURL: &wireRequestImageURL{
						URL: dataURL,
					},
				})
			default:
				return wireRequestMessage{}, fmt.Errorf("%s: unsupported message.content part type %T", provider, p)
			}
		}
		out.Content = wireRequestParts(parts)
		return out, nil
	}

	out.Content = wireRequestText("")
	return out, nil
}
