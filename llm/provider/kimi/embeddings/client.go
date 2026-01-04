package embeddings

import (
	"context"
	"net/http"
	"strings"

	"github.com/lgc202/go-kit/llm"
	openaiCompatEmbeddings "github.com/lgc202/go-kit/llm/internal/openai_compat/embeddings"
	"github.com/lgc202/go-kit/llm/schema"
)

const DefaultBaseURL = "https://api.moonshot.cn/v1"

var _ llm.Embedder = (*Client)(nil)
var _ llm.ProviderNamer = (*Client)(nil)

type Config struct {
	BaseURL    string
	APIKey     string
	HTTPClient *http.Client

	// DefaultHeaders are applied first, then overridden by request-level headers
	DefaultHeaders http.Header

	// DefaultOptions provides client-level defaults for request options
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
		Provider:       llm.ProviderKimi,
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

func (*Client) Provider() llm.Provider { return llm.ProviderKimi }

func (c *Client) Embed(ctx context.Context, inputs []string, opts ...llm.EmbeddingOption) (schema.EmbeddingResponse, error) {
	return c.inner.Embed(ctx, inputs, opts...)
}
