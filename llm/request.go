package llm

// RequestOption mutates a ChatRequest.
//
// Prefer passing options directly to Client.Chat/ChatStream; use BuildChatRequest
// only when you need to call Client.ChatRequest/ChatStreamRequest.
type RequestOption func(*ChatRequest)

func newChatRequest(model string, messages ...Message) ChatRequest {
	return ChatRequest{
		Model:    model,
		Messages: cloneMessages(messages),
		Extra:    map[string]any{},
	}
}

func applyOptions(req *ChatRequest, opts ...RequestOption) {
	if req == nil {
		return
	}
	for _, opt := range opts {
		if opt != nil {
			opt(req)
		}
	}
}

// BuildChatRequest creates a request from model + messages and applies opts.
func BuildChatRequest(model string, messages []Message, opts ...RequestOption) ChatRequest {
	req := newChatRequest(model, messages...)
	applyOptions(&req, opts...)
	return req
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

func WithHeader(key, value string) RequestOption {
	return func(r *ChatRequest) {
		if r.Transport == nil {
			r.Transport = &TransportOptions{}
		}
		if r.Transport.Headers == nil {
			r.Transport.Headers = make(map[string][]string)
		}
		r.Transport.Headers.Set(key, value)
	}
}

func cloneMessages(messages []Message) []Message {
	if messages == nil {
		return nil
	}
	out := make([]Message, len(messages))
	for i := range messages {
		out[i] = messages[i].Clone()
	}
	return out
}
