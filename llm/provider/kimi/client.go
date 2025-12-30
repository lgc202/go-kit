package kimi

import (
	"context"

	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/provider/openai"
	"github.com/lgc202/go-kit/llm/schema"
)

const DefaultBaseURL = "https://api.moonshot.cn/v1"

type Config = openai.Config

type Client struct {
	inner *openai.Client
}

var _ llm.ChatModel = (*Client)(nil)
var _ llm.ProviderNamer = (*Client)(nil)

func New(cfg Config) (*Client, error) {
	if cfg.BaseURL == "" {
		cfg.BaseURL = DefaultBaseURL
	}
	inner, err := openai.New(cfg)
	if err != nil {
		return nil, err
	}
	return &Client{inner: inner}, nil
}

func (*Client) Provider() llm.Provider { return llm.ProviderKimi }

func (c *Client) Chat(ctx context.Context, messages []schema.Message, opts ...llm.RequestOption) (schema.ChatResponse, error) {
	return c.inner.Chat(ctx, messages, opts...)
}

func (c *Client) ChatStream(ctx context.Context, messages []schema.Message, opts ...llm.RequestOption) (llm.Stream, error) {
	return c.inner.ChatStream(ctx, messages, opts...)
}
