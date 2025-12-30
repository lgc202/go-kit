package openai_compat

import (
	"strings"

	"github.com/lgc202/go-kit/llm"
)

func (p *Provider) mapRequest(req llm.ChatRequest) (map[string]any, error) {
	if p.hooks.BeforeMap != nil {
		p.hooks.BeforeMap(&req)
	}

	model := req.Model
	if model == "" {
		model = p.model
	}

	wmessages := make([]apiMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		wm := apiMessage{Role: string(m.Role), Name: m.Name}
		content, err := p.mapMessageContent(m)
		if err != nil {
			return nil, err
		}
		wm.Content = content
		if m.Role == llm.RoleTool {
			wm.ToolCallID = m.ToolCallID
		}
		if len(m.ToolCalls) > 0 {
			wm.ToolCalls = make([]apiToolCall, 0, len(m.ToolCalls))
			for i, tc := range m.ToolCalls {
				wm.ToolCalls = append(wm.ToolCalls, apiToolCall{
					Index: i,
					ID:    tc.ID,
					Type:  "function",
					Function: apiFunctionCall{
						Name:      tc.Name,
						Arguments: firstNonEmpty(tc.ArgumentsText, string(tc.Arguments)),
					},
				})
			}
		}
		wmessages = append(wmessages, wm)
	}

	wtools := make([]apiTool, 0, len(req.Tools))
	for _, t := range req.Tools {
		wtools = append(wtools, apiTool{
			Type: "function",
			Function: apiFunctionDef{
				Name:        t.Name,
				Description: t.Description,
				Parameters:  t.InputSchema,
			},
		})
	}

	m := map[string]any{
		"model":    model,
		"messages": wmessages,
	}

	if req.Temperature != nil {
		m["temperature"] = *req.Temperature
	}
	if req.TopP != nil {
		m["top_p"] = *req.TopP
	}
	if req.MaxTokens != nil {
		m["max_tokens"] = *req.MaxTokens
	}
	if req.Seed != nil {
		m["seed"] = *req.Seed
	}
	if req.PresencePenalty != nil {
		m["presence_penalty"] = *req.PresencePenalty
	}
	if req.FrequencyPenalty != nil {
		m["frequency_penalty"] = *req.FrequencyPenalty
	}
	if len(req.Stop) > 0 {
		m["stop"] = req.Stop
	}
	if req.LogProbs != nil {
		m["logprobs"] = *req.LogProbs
	}
	if req.TopLogProbs != nil {
		m["top_logprobs"] = *req.TopLogProbs
	}
	if req.ResponseFormat != nil {
		m["response_format"] = req.ResponseFormat
	}
	if req.StreamOptions != nil {
		m["stream_options"] = req.StreamOptions
	}

	if len(wtools) > 0 {
		m["tools"] = wtools
	}
	if req.ToolChoice != nil {
		m["tool_choice"] = mapToolChoice(*req.ToolChoice)
	}

	for k, v := range req.Extra {
		m[k] = v
	}
	if p.hooks.PatchRequest != nil {
		p.hooks.PatchRequest(m)
	}
	return m, nil
}

func mapToolChoice(tc llm.ToolChoice) any {
	switch tc.Mode {
	case llm.ToolChoiceNone:
		return "none"
	case llm.ToolChoiceRequired:
		return "required"
	case llm.ToolChoiceFunction:
		return map[string]any{
			"type": "function",
			"function": map[string]any{
				"name": tc.FunctionName,
			},
		}
	case llm.ToolChoiceAuto:
		fallthrough
	default:
		return "auto"
	}
}

func (p *Provider) mapMessageContent(msg llm.Message) (any, error) {
	if p.hooks.MapMessageContent != nil {
		v, err := p.hooks.MapMessageContent(msg)
		if err != nil {
			return nil, err
		}
		if v != nil {
			return v, nil
		}
	}

	// Tool outputs are mapped to a plain string.
	if msg.Role == llm.RoleTool {
		return flattenPartsText(msg.Parts), nil
	}
	return mapPartsToWireContent(p.name, msg.Parts)
}

func mapPartsToWireContent(provider string, parts []llm.ContentPart) (any, error) {
	if len(parts) == 0 {
		return "", nil
	}

	// Prefer the simplest representation when possible.
	if len(parts) == 1 && parts[0].Type == llm.ContentPartText && parts[0].JSON == nil && len(parts[0].Data) == 0 {
		return parts[0].Text, nil
	}

	out := make([]any, 0, len(parts))
	for _, p := range parts {
		switch p.Type {
		case llm.ContentPartText, llm.ContentPartReasoning:
			// Most OpenAI-compatible providers expect "text" parts; reasoning is usually
			// a response-side feature. Treat reasoning parts as text when sending.
			out = append(out, map[string]any{"type": "text", "text": p.Text})
		default:
			return nil, &llm.LLMError{
				Provider: provider,
				Kind:     llm.ErrKindBadRequest,
				Message:  "unsupported message content part type: " + string(p.Type),
			}
		}
	}
	return out, nil
}

func flattenPartsText(parts []llm.ContentPart) string {
	var b strings.Builder
	for _, p := range parts {
		if p.Text == "" {
			continue
		}
		b.WriteString(p.Text)
	}
	return b.String()
}
