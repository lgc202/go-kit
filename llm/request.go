package llm

// RequestOption mutates a ChatRequest.
//
// Use with ChatRequest.With(...) or in your own builder.
type RequestOption func(*ChatRequest)

func NewChatRequest(model string, messages ...Message) ChatRequest {
	return ChatRequest{
		Model:    model,
		Messages: append([]Message(nil), messages...),
		Extra:    map[string]any{},
	}
}

func (r ChatRequest) With(opts ...RequestOption) ChatRequest {
	out := r.Clone()
	for _, opt := range opts {
		if opt != nil {
			opt(&out)
		}
	}
	return out
}

func WithModel(model string) RequestOption {
	return func(r *ChatRequest) { r.Model = model }
}

func WithTemperature(v float64) RequestOption {
	return func(r *ChatRequest) { r.Temperature = &v }
}

func WithTopP(v float64) RequestOption {
	return func(r *ChatRequest) { r.TopP = &v }
}

func WithMaxTokens(v int) RequestOption {
	return func(r *ChatRequest) { r.MaxTokens = &v }
}

func WithSeed(v int64) RequestOption {
	return func(r *ChatRequest) { r.Seed = &v }
}

func WithPresencePenalty(v float64) RequestOption {
	return func(r *ChatRequest) { r.PresencePenalty = &v }
}

func WithFrequencyPenalty(v float64) RequestOption {
	return func(r *ChatRequest) { r.FrequencyPenalty = &v }
}

func WithStop(stop ...string) RequestOption {
	return func(r *ChatRequest) {
		if stop == nil {
			r.Stop = nil
			return
		}
		r.Stop = append([]string(nil), stop...)
	}
}

func WithResponseFormatText() RequestOption {
	return func(r *ChatRequest) { r.ResponseFormat = &ResponseFormat{Type: ResponseFormatText} }
}

func WithResponseFormatJSONObject() RequestOption {
	return func(r *ChatRequest) { r.ResponseFormat = &ResponseFormat{Type: ResponseFormatJSONObject} }
}

func WithResponseFormatJSONSchema(schemaJSON []byte) RequestOption {
	return func(r *ChatRequest) {
		r.ResponseFormat = &ResponseFormat{Type: ResponseFormatJSONSchema, JSONSchema: append([]byte(nil), schemaJSON...)}
	}
}

func WithLogProbs(enabled bool) RequestOption {
	return func(r *ChatRequest) { r.LogProbs = &enabled }
}

func WithTopLogProbs(v int) RequestOption {
	return func(r *ChatRequest) { r.TopLogProbs = &v }
}

func WithStreamIncludeUsage(enabled bool) RequestOption {
	return func(r *ChatRequest) { r.StreamOptions = &StreamOptions{IncludeUsage: enabled} }
}

func WithTools(tools ...ToolDefinition) RequestOption {
	return func(r *ChatRequest) { r.Tools = append([]ToolDefinition(nil), tools...) }
}

func WithToolChoice(choice ToolChoice) RequestOption {
	return func(r *ChatRequest) { r.ToolChoice = &choice }
}

func WithExtra(key string, value any) RequestOption {
	return func(r *ChatRequest) {
		if r.Extra == nil {
			r.Extra = make(map[string]any)
		}
		r.Extra[key] = value
	}
}
