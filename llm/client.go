package llm

import "context"

// Client is a provider-agnostic LLM SDK entrypoint.
type Client struct {
	provider Provider
}

func New(provider Provider) *Client {
	return &Client{provider: provider}
}

func (c *Client) Chat(ctx context.Context, req ChatRequest) (ChatResponse, error) {
	return c.provider.Chat(ctx, req)
}

func (c *Client) ChatStream(ctx context.Context, req ChatRequest) (Stream, error) {
	return c.provider.ChatStream(ctx, req)
}

func (c *Client) Provider() Provider {
	return c.provider
}
