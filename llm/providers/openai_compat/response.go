package openai_compat

import (
	"strings"

	"github.com/lgc202/go-kit/llm"
)

func (p *Provider) mapResponse(r chatCompletionResponse) llm.ChatResponse {
	out := llm.ChatResponse{
		ID:      r.ID,
		Model:   r.Model,
		Created: r.CreatedTime(),
		Choices: make([]llm.ChatChoice, 0, len(r.Choices)),
	}
	if r.Usage != nil {
		u := &llm.Usage{
			PromptTokens:     r.Usage.PromptTokens,
			CompletionTokens: r.Usage.CompletionTokens,
			TotalTokens:      r.Usage.TotalTokens,
		}
		d := llm.UsageDetails{
			PromptCacheHitTokens:  r.Usage.intField("prompt_cache_hit_tokens"),
			PromptCacheMissTokens: r.Usage.intField("prompt_cache_miss_tokens"),
		}
		cachedTokens := r.Usage.intField("cached_tokens")
		if d.PromptCacheHitTokens == 0 && cachedTokens != 0 {
			// Some providers only report a single cache number.
			d.PromptCacheHitTokens = cachedTokens
		}
		d.ReasoningTokens = r.Usage.intFieldInObject("completion_tokens_details", "reasoning_tokens")
		if d.PromptCacheHitTokens != 0 || d.PromptCacheMissTokens != 0 || d.ReasoningTokens != 0 {
			u.Details = &d
		}
		out.Usage = u
	}

	for _, c := range r.Choices {
		msg := llm.Message{Role: llm.RoleAssistant}
		if c.Message.Role != "" {
			msg.Role = llm.Role(c.Message.Role)
		}
		text, reasoningFromContent := splitContent(c.Message.Content)
		reasoning := reasoningFromContent
		if c.Message.ReasoningContent != "" {
			reasoning = c.Message.ReasoningContent + reasoning
		}
		if thinking := anyString(c.Message.Thinking); thinking != "" {
			reasoning = thinking + reasoning
		}
		if text != "" {
			msg.Parts = append(msg.Parts, llm.TextPart(text))
		}
		if reasoning != "" {
			msg.Parts = append(msg.Parts, llm.ReasoningPart(reasoning))
		}
		msg.Name = c.Message.Name
		if len(c.Message.ToolCalls) > 0 {
			msg.ToolCalls = make([]llm.ToolCall, 0, len(c.Message.ToolCalls))
			for _, tc := range c.Message.ToolCalls {
				msg.ToolCalls = append(msg.ToolCalls, llm.ToolCall{
					ID:            tc.ID,
					Name:          tc.Function.Name,
					ArgumentsText: tc.Function.Arguments,
				})
			}
		}
		out.Choices = append(out.Choices, llm.ChatChoice{
			Index:        c.Index,
			Message:      msg,
			FinishReason: mapFinishReason(c.FinishReason),
		})
	}
	return out
}

func mapFinishReason(fr string) llm.FinishReason {
	switch fr {
	case "stop":
		return llm.FinishReasonStop
	case "length":
		return llm.FinishReasonLength
	case "tool_calls", "function_call":
		return llm.FinishReasonToolCalls
	case "":
		return ""
	default:
		return llm.FinishReasonUnknown
	}
}

func contentText(v any) string {
	text, _ := splitContent(v)
	return text
}

func splitContent(v any) (text string, reasoning string) {
	switch x := v.(type) {
	case nil:
		return "", ""
	case string:
		return x, ""
	case []any:
		var b strings.Builder
		var r strings.Builder
		for _, it := range x {
			if m, ok := it.(map[string]any); ok {
				typeStr, _ := m["type"].(string)
				if t, ok := m["text"].(string); ok {
					switch typeStr {
					case "reasoning", "thinking":
						r.WriteString(t)
					default:
						b.WriteString(t)
					}
				}
			}
		}
		return b.String(), r.String()
	case map[string]any:
		typeStr, _ := x["type"].(string)
		if t, ok := x["text"].(string); ok {
			switch typeStr {
			case "reasoning", "thinking":
				return "", t
			default:
				return t, ""
			}
		}
		return "", ""
	default:
		return "", ""
	}
}

func anyString(v any) string {
	s, _ := v.(string)
	return s
}
