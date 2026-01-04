package embeddings

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"
	"slices"
	"strings"

	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/internal/openai_compat/transport"
	"github.com/lgc202/go-kit/llm/schema"
)

const httpAcceptJSON = "application/json"

// DefaultPath is the OpenAI-compatible embeddings endpoint path.
const DefaultPath = "/embeddings"

type Config struct {
	Provider llm.Provider

	BaseURL string
	Path    string

	APIKey     string
	HTTPClient *http.Client

	DefaultHeaders http.Header

	// DefaultOptions provides client-level defaults for request options.
	DefaultOptions []llm.EmbeddingOption
}

type Client struct {
	provider string

	t *transport.Client

	defaultOpts []llm.EmbeddingOption
}

var _ llm.Embedder = (*Client)(nil)

func New(cfg Config) (*Client, error) {
	t, err := transport.New(transport.Config{
		Provider:       cfg.Provider,
		BaseURL:        cfg.BaseURL,
		Path:           cfg.Path,
		DefaultPath:    DefaultPath,
		APIKey:         cfg.APIKey,
		HTTPClient:     cfg.HTTPClient,
		DefaultHeaders: cfg.DefaultHeaders,
	})
	if err != nil {
		return nil, err
	}

	return &Client{
		provider:    t.Provider(),
		t:           t,
		defaultOpts: slices.Clone(cfg.DefaultOptions),
	}, nil
}

func (c *Client) Embed(ctx context.Context, inputs []string, opts ...llm.EmbeddingOption) (schema.EmbeddingResponse, error) {
	reqCfg := llm.ApplyEmbeddingOptions(slices.Concat(c.defaultOpts, opts)...)

	if len(inputs) == 0 {
		return schema.EmbeddingResponse{}, fmt.Errorf("%s: inputs required", c.provider)
	}
	if strings.TrimSpace(reqCfg.Model) == "" {
		return schema.EmbeddingResponse{}, fmt.Errorf("%s: model required (use llm.WithModel)", c.provider)
	}

	req := embeddingRequest{
		provider: c.provider,
		Model:    reqCfg.Model,
		Input:    slices.Clone(inputs),
		User:     reqCfg.User,
	}
	req.extra = reqCfg.ExtraFields
	req.allowExtraFieldOverride = reqCfg.AllowExtraFieldOverride

	resp, err := c.t.PostJSON(ctx, req, transport.RequestConfig{
		Timeout:    reqCfg.Timeout,
		Headers:    reqCfg.Headers,
		ErrorHooks: reqCfg.ErrorHooks,
	}, httpAcceptJSON)
	if err != nil {
		return schema.EmbeddingResponse{}, err
	}
	defer resp.Body.Close()

	if reqCfg.KeepRaw {
		b, rerr := io.ReadAll(resp.Body)
		if rerr != nil {
			return schema.EmbeddingResponse{}, fmt.Errorf("%s: read response: %w", c.provider, rerr)
		}
		return c.mapEmbeddingResponseBytes(b, true)
	}

	var in embeddingResponse
	if err := json.NewDecoder(resp.Body).Decode(&in); err != nil {
		return schema.EmbeddingResponse{}, fmt.Errorf("%s: decode response: %w", c.provider, err)
	}
	return toSchemaEmbeddingResponse(in), nil
}

func (c *Client) mapEmbeddingResponseBytes(raw []byte, keepRaw bool) (schema.EmbeddingResponse, error) {
	var in embeddingResponse
	if err := json.Unmarshal(raw, &in); err != nil {
		return schema.EmbeddingResponse{}, fmt.Errorf("%s: decode response: %w", c.provider, err)
	}
	out := toSchemaEmbeddingResponse(in)
	if keepRaw {
		out.Raw = json.RawMessage(raw)
	}
	return out, nil
}
