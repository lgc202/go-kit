package openai_compat

import (
	"context"
	"encoding/json"
	"errors"
	"log/slog"
	"net/http"

	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/internal/transport"
)

type Provider struct {
	name string

	apiKey string
	model  string
	path   string

	tr    *transport.Client
	hooks Hooks
}

func New(apiKey string, opts ...Option) (*Provider, error) {
	tr, err := transport.New("https://api.openai.com", nil)
	if err != nil {
		return nil, err
	}

	p := &Provider{
		name:   "openai_compat",
		apiKey: apiKey,
		path:   "/v1/chat/completions",
		tr:     tr,
	}

	for _, opt := range opts {
		if err := opt(p); err != nil {
			return nil, err
		}
	}

	if p.tr == nil {
		return nil, errors.New("openai_compat: nil transport")
	}
	if p.tr.Logger == nil {
		p.tr.Logger = slog.Default()
	}

	return p, nil
}

func WithProviderName(name string) Option {
	return func(p *Provider) error {
		p.name = name
		return nil
	}
}

func WithBaseURL(baseURL string) Option {
	return func(p *Provider) error {
		tr, err := transport.New(baseURL, p.tr.HTTPClient)
		if err != nil {
			return err
		}
		tr.DefaultHeaders = p.tr.DefaultHeaders.Clone()
		tr.UserAgent = p.tr.UserAgent
		tr.Logger = p.tr.Logger
		tr.Retry = p.tr.Retry
		p.tr = tr
		return nil
	}
}

func WithHTTPClient(c *http.Client) Option {
	return func(p *Provider) error {
		p.tr.HTTPClient = c
		return nil
	}
}

func WithUserAgent(ua string) Option {
	return func(p *Provider) error {
		p.tr.UserAgent = ua
		return nil
	}
}

func WithLogger(logger *slog.Logger) Option {
	return func(p *Provider) error {
		if logger != nil {
			p.tr.Logger = logger
		}
		return nil
	}
}

func WithRetry(cfg transport.RetryConfig) Option {
	return func(p *Provider) error {
		p.tr.Retry = cfg
		return nil
	}
}

func WithDefaultHeader(key, value string) Option {
	return func(p *Provider) error {
		p.tr.DefaultHeaders.Add(key, value)
		return nil
	}
}

func WithChatCompletionsPath(path string) Option {
	return func(p *Provider) error {
		p.path = path
		return nil
	}
}

func WithDefaultModel(model string) Option {
	return func(p *Provider) error {
		p.model = model
		return nil
	}
}

func (p *Provider) Chat(ctx context.Context, req llm.ChatRequest) (llm.ChatResponse, error) {
	if err := p.validateRequest(req); err != nil {
		return llm.ChatResponse{}, err
	}

	wreq := p.mapRequest(req)
	hdr := p.defaultHeaders("application/json")

	_, raw, err := p.tr.DoJSON(ctx, http.MethodPost, p.path, hdr, wreq)
	if err != nil {
		return llm.ChatResponse{}, p.mapError(err, raw)
	}

	var wresp chatCompletionResponse
	if err := json.Unmarshal(raw, &wresp); err != nil {
		return llm.ChatResponse{}, &llm.LLMError{Provider: p.name, Kind: llm.ErrKindParse, Message: "failed to decode response", Raw: raw, Cause: err}
	}

	out := p.mapResponse(wresp)
	out.RawJSON = append([]byte(nil), raw...)
	return out, nil
}

func (p *Provider) ChatStream(ctx context.Context, req llm.ChatRequest) (llm.Stream, error) {
	if err := p.validateRequest(req); err != nil {
		return nil, err
	}

	wreq := p.mapRequest(req)
	wreq["stream"] = true

	hdr := p.defaultHeaders("text/event-stream")
	resp, err := p.tr.DoStream(ctx, http.MethodPost, p.path, hdr, wreq)
	if err != nil {
		var se *transport.HTTPStatusError
		if errors.As(err, &se) {
			return nil, p.mapError(err, se.Body)
		}
		return nil, p.mapError(err, nil)
	}

	return newStream(p.name, resp), nil
}

func (p *Provider) defaultHeaders(accept string) http.Header {
	h := make(http.Header)
	h.Set("Content-Type", "application/json")
	if accept != "" {
		h.Set("Accept", accept)
	}
	if p.apiKey != "" {
		h.Set("Authorization", "Bearer "+p.apiKey)
	}
	if p.hooks.PatchHeaders != nil {
		p.hooks.PatchHeaders(h)
	}
	return h
}

func (p *Provider) validateRequest(req llm.ChatRequest) error {
	if req.Model == "" && p.model == "" {
		return errors.New("llm: model is required")
	}
	if len(req.Messages) == 0 {
		return errors.New("llm: messages is required")
	}
	return nil
}
