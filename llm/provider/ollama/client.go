package ollama

import (
	"context"
	"net/http"
	"strings"

	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/internal/openai_compat"
	"github.com/lgc202/go-kit/llm/schema"
)

// Ollama 的 OpenAI 兼容端点（通常为本地地址）
const DefaultBaseURL = "http://localhost:11434/v1"

type Config struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client

	// DefaultHeaders 首先应用，然后被请求级别的 headers 覆盖
	DefaultHeaders http.Header

	// DefaultRequest 提供客户端级别的默认请求选项
	DefaultRequest llm.RequestConfig
}

type Client struct {
	inner *openai_compat.Client
}

var _ llm.ChatModel = (*Client)(nil)
var _ llm.ProviderNamer = (*Client)(nil)

func New(cfg Config) (*Client, error) {
	base := strings.TrimSpace(cfg.BaseURL)
	if base == "" {
		base = DefaultBaseURL
	}

	inner, err := openai_compat.New(openai_compat.Config{
		Provider:       llm.ProviderOllama,
		BaseURL:        base,
		Path:           "/chat/completions",
		APIKey:         cfg.APIKey,
		HTTPClient:     cfg.HTTPClient,
		DefaultHeaders: cfg.DefaultHeaders,
		DefaultRequest: cfg.DefaultRequest,
		Adapter:        adapter{},
	})
	if err != nil {
		return nil, err
	}

	return &Client{inner: inner}, nil
}

func (*Client) Provider() llm.Provider { return llm.ProviderOllama }

func (c *Client) Chat(ctx context.Context, messages []schema.Message, opts ...llm.RequestOption) (schema.ChatResponse, error) {
	return c.inner.Chat(ctx, messages, opts...)
}

func (c *Client) ChatStream(ctx context.Context, messages []schema.Message, opts ...llm.RequestOption) (llm.Stream, error) {
	return c.inner.ChatStream(ctx, messages, opts...)
}
