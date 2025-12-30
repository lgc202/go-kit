package llm

import "context"

// Client is a provider-agnostic LLM SDK entrypoint.
type Client struct {
	provider    Provider
	defaultOpts []RequestOption
}

// New creates a client with optional default request options.
//
// Default request options are applied to every call, and per-call options are applied
// after defaults (so per-call overrides win).
func New(provider Provider, defaultOpts ...RequestOption) *Client {
	return &Client{provider: provider, defaultOpts: append([]RequestOption(nil), defaultOpts...)}
}

// Chat sends a chat completion request.
//
// Request parameters should be provided via options.
func (c *Client) Chat(ctx context.Context, messages []Message, opts ...RequestOption) (ChatResponse, error) {
	req := c.buildRequest(messages, opts...)
	return c.provider.Chat(ctx, req)
}

// ChatStream sends a streaming chat completion request.
func (c *Client) ChatStream(ctx context.Context, messages []Message, opts ...RequestOption) (Stream, error) {
	req := c.buildRequest(messages, opts...)
	return c.provider.ChatStream(ctx, req)
}

// ChatRequest sends a fully-specified request. Use this only when you need full control.
func (c *Client) ChatRequest(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	return c.provider.Chat(ctx, req)
}

// ChatStreamRequest sends a fully-specified streaming request.
func (c *Client) ChatStreamRequest(ctx context.Context, req ChatRequest) (Stream, error) {
	return c.provider.ChatStream(ctx, req)
}

func (c *Client) Provider() Provider {
	return c.provider
}

func (c *Client) buildRequest(messages []Message, opts ...RequestOption) ChatRequest {
	req := ChatRequest{Messages: cloneMessages(messages), Extra: map[string]any{}}
	for _, opt := range c.defaultOpts {
		if opt != nil {
			opt(&req)
		}
	}
	for _, opt := range opts {
		if opt != nil {
			opt(&req)
		}
	}
	return req
}
