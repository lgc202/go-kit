package embeddings

import (
	"context"
	"net/http"
	"strings"

	"github.com/lgc202/go-kit/llm"
	openaiCompatEmbeddings "github.com/lgc202/go-kit/llm/internal/openai_compat/embeddings"
	"github.com/lgc202/go-kit/llm/schema"
)

// DashScope OpenAI 兼容端点
const DefaultBaseURL = "https://dashscope.aliyuncs.com/compatible-mode/v1"

type Config struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client

	// DefaultHeaders 首先应用，然后被请求级别的 headers 覆盖
	DefaultHeaders http.Header

	// DefaultOptions 提供客户端级别的默认请求选项
	DefaultOptions []llm.EmbeddingOption
}

type Client struct {
	inner *openaiCompatEmbeddings.Client
}

var _ llm.Embedder = (*Client)(nil)
var _ llm.ProviderNamer = (*Client)(nil)

func New(cfg Config) (*Client, error) {
	base := strings.TrimSpace(cfg.BaseURL)
	if base == "" {
		base = DefaultBaseURL
	}

	inner, err := openaiCompatEmbeddings.New(openaiCompatEmbeddings.Config{
		Provider:       llm.ProviderQwen,
		BaseURL:        base,
		Path:           "/embeddings",
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

func (c *Client) Embed(ctx context.Context, inputs []string, opts ...llm.EmbeddingOption) (schema.EmbeddingResponse, error) {
	return c.inner.Embed(ctx, inputs, opts...)
}
