package deepseek

import (
	"context"
	"net/http"
	"strings"

	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/internal/openai_compat"
	"github.com/lgc202/go-kit/llm/schema"
)

const DefaultBaseURL = "https://api.deepseek.com"

var _ llm.ChatModel = (*Client)(nil)
var _ llm.ProviderNamer = (*Client)(nil)

type Config struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client

	// DefaultHeaders are applied first, then overridden by request-level headers.
	DefaultHeaders http.Header

	// DefaultOptions provides client-level defaults for request options.
	DefaultOptions []llm.RequestOption
}

type Client struct {
	inner *openai_compat.Client
}

func New(cfg Config) (*Client, error) {
	base := strings.TrimSpace(cfg.BaseURL)
	if base == "" {
		base = DefaultBaseURL
	}

	inner, err := openai_compat.New(openai_compat.Config{
		Provider:       llm.ProviderDeepSeek,
		BaseURL:        base,
		Path:           "/chat/completions",
		APIKey:         cfg.APIKey,
		HTTPClient:     cfg.HTTPClient,
		DefaultHeaders: cfg.DefaultHeaders,
		DefaultOptions: cfg.DefaultOptions,
	})
	if err != nil {
		return nil, err
	}

	return &Client{inner: inner}, nil
}

func (*Client) Provider() llm.Provider { return llm.ProviderDeepSeek }

func (c *Client) Chat(ctx context.Context, messages []schema.Message, opts ...llm.RequestOption) (schema.ChatResponse, error) {
	return c.inner.Chat(ctx, messages, opts...)
}

func (c *Client) ChatStream(ctx context.Context, messages []schema.Message, opts ...llm.RequestOption) (llm.Stream, error) {
	return c.inner.ChatStream(ctx, messages, opts...)
}
