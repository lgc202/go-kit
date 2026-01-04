package chat

import (
	"context"
	"strings"

	"github.com/lgc202/go-kit/llm"
	openaiCompatChat "github.com/lgc202/go-kit/llm/internal/openai_compat/chat"
	"github.com/lgc202/go-kit/llm/provider/base"
	"github.com/lgc202/go-kit/llm/schema"
)

// DefaultBaseURL DashScope OpenAI 兼容端点
const DefaultBaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"

var _ llm.ChatModel = (*Client)(nil)
var _ llm.ProviderNamer = (*Client)(nil)

type BaseConfig = base.Config

type Config struct {
	BaseConfig

	// DefaultOptions 客户端级别的默认请求选项
	DefaultOptions []llm.ChatOption
}

type Client struct {
	inner *openaiCompatChat.Client
}

func New(cfg Config) (*Client, error) {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	inner, err := openaiCompatChat.New(openaiCompatChat.Config{
		Provider:       llm.ProviderQwen,
		BaseURL:        baseURL,
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

func (*Client) Provider() llm.Provider { return llm.ProviderQwen }

func (c *Client) Chat(ctx context.Context, messages []schema.Message, opts ...llm.ChatOption) (schema.ChatResponse, error) {
	return c.inner.Chat(ctx, messages, opts...)
}

func (c *Client) ChatStream(ctx context.Context, messages []schema.Message, opts ...llm.ChatOption) (llm.Stream, error) {
	return c.inner.ChatStream(ctx, messages, opts...)
}
