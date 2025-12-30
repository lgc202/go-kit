package llm

import "context"

// Provider is the minimal interface an LLM backend must implement.
//
// Implementations are expected to:
// - treat Request as read-only
// - return an LLMError (or wrap one) for provider/HTTP errors
// - honor ctx cancellation
type Provider interface {
	Chat(ctx context.Context, req ChatRequest) (ChatResponse, error)
	ChatStream(ctx context.Context, req ChatRequest) (Stream, error)
}
