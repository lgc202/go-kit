package chat

import (
	"context"
	"net/http"
	"strings"

	"github.com/lgc202/go-kit/llm"
	openaiCompatChat "github.com/lgc202/go-kit/llm/internal/openai_compat/chat"
	"github.com/lgc202/go-kit/llm/schema"
)

// DefaultBaseURL DashScope OpenAI 兼容端点
const DefaultBaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"

type Config struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client

	// DefaultHeaders 默认请求头，会被请求级别的 headers 覆盖
	DefaultHeaders http.Header

	// DefaultOptions 客户端级别的默认请求选项
	DefaultOptions []llm.ChatOption
}

type Client struct {
	inner *openaiCompatChat.Client
}

var _ llm.ChatModel = (*Client)(nil)
var _ llm.ProviderNamer = (*Client)(nil)

func New(cfg Config) (*Client, error) {
	base := strings.TrimSpace(cfg.BaseURL)
	if base == "" {
		base = DefaultBaseURL
	}

	inner, err := openaiCompatChat.New(openaiCompatChat.Config{
		Provider:       llm.ProviderQwen,
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

func (*Client) Provider() llm.Provider { return llm.ProviderQwen }

func (c *Client) Chat(ctx context.Context, messages []schema.Message, opts ...llm.ChatOption) (schema.ChatResponse, error) {
	return c.inner.Chat(ctx, messages, opts...)
}

func (c *Client) ChatStream(ctx context.Context, messages []schema.Message, opts ...llm.ChatOption) (llm.Stream, error) {
	return c.inner.ChatStream(ctx, messages, opts...)
}
