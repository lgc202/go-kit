package embeddings

import (
	"context"
	"net/http"
	"strings"

	"github.com/lgc202/go-kit/llm"
	openaiCompatEmbeddings "github.com/lgc202/go-kit/llm/internal/openai_compat/embeddings"
	"github.com/lgc202/go-kit/llm/schema"
)

const DefaultBaseURL = "https://api.openai.com/v1"

var _ llm.Embedder = (*Client)(nil)
var _ llm.ProviderNamer = (*Client)(nil)

type Config struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client

	// DefaultHeaders 默认请求头，会被请求级别的 headers 覆盖
	DefaultHeaders http.Header

	// DefaultOptions 客户端级别的默认请求选项
	DefaultOptions []llm.EmbeddingOption
}

type Client struct {
	inner *openaiCompatEmbeddings.Client
}

func New(cfg Config) (*Client, error) {
	base := strings.TrimSpace(cfg.BaseURL)
	if base == "" {
		base = DefaultBaseURL
	}

	inner, err := openaiCompatEmbeddings.New(openaiCompatEmbeddings.Config{
		Provider:       llm.ProviderOpenAI,
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

func (*Client) Provider() llm.Provider { return llm.ProviderOpenAI }

func (c *Client) Embed(ctx context.Context, inputs []string, opts ...llm.EmbeddingOption) (schema.EmbeddingResponse, error) {
	return c.inner.Embed(ctx, inputs, opts...)
}
