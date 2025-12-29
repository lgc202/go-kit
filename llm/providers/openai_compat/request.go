package openai_compat

import "github.com/lgc202/go-kit/llm"

func (p *Provider) mapRequest(req llm.ChatRequest) map[string]any {
	if p.hooks.BeforeMap != nil {
		p.hooks.BeforeMap(&req)
	}

	model := req.Model
	if model == "" {
		model = p.model
	}

	wmessages := make([]wireMessage, 0, len(req.Messages))
	for _, m := range req.Messages {
		wm := wireMessage{Role: string(m.Role), Name: m.Name}
		wm.Content = m.Content
		if m.Role == llm.RoleTool {
			wm.ToolCallID = m.ToolCallID
		}
		if len(m.ToolCalls) > 0 {
			wm.ToolCalls = make([]wireToolCall, 0, len(m.ToolCalls))
			for i, tc := range m.ToolCalls {
				wm.ToolCalls = append(wm.ToolCalls, wireToolCall{
					Index: i,
					ID:    tc.ID,
					Type:  "function",
					Function: wireFunctionCall{
						Name:      tc.Name,
						Arguments: firstNonEmpty(tc.ArgumentsText, string(tc.Arguments)),
					},
				})
			}
		}
		wmessages = append(wmessages, wm)
	}

	wtools := make([]wireTool, 0, len(req.Tools))
	for _, t := range req.Tools {
		wtools = append(wtools, wireTool{
			Type: "function",
			Function: wireFunctionDef{
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
	return m
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
