package embeddings

import (
	"context"
	"strings"

	"github.com/lgc202/go-kit/llm"
	openaiCompatEmbeddings "github.com/lgc202/go-kit/llm/internal/openai_compat/embeddings"
	"github.com/lgc202/go-kit/llm/provider/base"
	"github.com/lgc202/go-kit/llm/schema"
)

const DefaultBaseURL = "https://api.moonshot.cn/v1"

var _ llm.Embedder = (*Client)(nil)
var _ llm.ProviderNamer = (*Client)(nil)

type BaseConfig = base.Config

type Config struct {
	BaseConfig

	// DefaultOptions 客户端级别的默认请求选项
	DefaultOptions []llm.EmbeddingOption
}

type Client struct {
	inner *openaiCompatEmbeddings.Client
}

func New(cfg Config) (*Client, error) {
	baseURL := strings.TrimSpace(cfg.BaseURL)
	if baseURL == "" {
		baseURL = DefaultBaseURL
	}

	inner, err := openaiCompatEmbeddings.New(openaiCompatEmbeddings.Config{
		Provider:       llm.ProviderKimi,
		BaseURL:        baseURL,
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

func (*Client) Provider() llm.Provider { return llm.ProviderKimi }

func (c *Client) Embed(ctx context.Context, inputs []string, opts ...llm.EmbeddingOption) (schema.EmbeddingResponse, error) {
	return c.inner.Embed(ctx, inputs, opts...)
}
