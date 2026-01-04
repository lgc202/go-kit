package chat

import (
	"bytes"
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

const (
	httpAcceptJSON = "application/json"
	httpAcceptSSE  = "text/event-stream"
)

// DefaultPath OpenAI 兼容的 chat completions 端点路径
const DefaultPath = "/chat/completions"

type Config struct {
	Provider llm.Provider

	BaseURL string
	Path    string

	APIKey     string
	HTTPClient *http.Client

	DefaultHeaders http.Header

	// DefaultOptions 客户端级别的默认请求选项
	DefaultOptions []llm.ChatOption
}

type Client struct {
	provider string

	t *transport.Client

	defaultOpts []llm.ChatOption
}

var _ llm.ChatModel = (*Client)(nil)

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

func (c *Client) Chat(ctx context.Context, messages []schema.Message, opts ...llm.ChatOption) (schema.ChatResponse, error) {
	reqCfg := llm.ApplyChatOptions(slices.Concat(c.defaultOpts, opts)...)

	payload, err := c.buildChatRequest(messages, reqCfg, false)
	if err != nil {
		return schema.ChatResponse{}, err
	}

	resp, err := c.t.PostJSON(ctx, payload, transport.RequestConfig{
		Timeout:    reqCfg.Timeout,
		Headers:    reqCfg.Headers,
		ErrorHooks: reqCfg.ErrorHooks,
	}, httpAcceptJSON)
	if err != nil {
		return schema.ChatResponse{}, err
	}
	defer resp.Body.Close()

	return c.parseChatResponse(resp.Body, reqCfg)
}

func (c *Client) ChatStream(ctx context.Context, messages []schema.Message, opts ...llm.ChatOption) (llm.Stream, error) {
	reqCfg := llm.ApplyChatOptions(slices.Concat(c.defaultOpts, opts)...)

	payload, err := c.buildChatRequest(messages, reqCfg, true)
	if err != nil {
		return nil, err
	}

	resp, err := c.t.PostJSON(ctx, payload, transport.RequestConfig{
		Timeout:    reqCfg.Timeout,
		Headers:    reqCfg.Headers,
		ErrorHooks: reqCfg.ErrorHooks,
	}, httpAcceptSSE)
	if err != nil {
		return nil, err
	}

	return newStream(c.provider, resp.Body, reqCfg.KeepRaw, reqCfg.StreamEventHooks), nil
}

func (c *Client) parseChatResponse(body io.Reader, cfg llm.ChatConfig) (schema.ChatResponse, error) {
	if cfg.KeepRaw || len(cfg.ResponseHooks) > 0 {
		respBytes, err := io.ReadAll(body)
		if err != nil {
			return schema.ChatResponse{}, fmt.Errorf("%s: read response: %w", c.provider, err)
		}
		return c.mapChatResponseBytes(respBytes, cfg.KeepRaw, cfg.ResponseHooks)
	}

	var in chatCompletionResponse
	if err := json.NewDecoder(body).Decode(&in); err != nil {
		return schema.ChatResponse{}, fmt.Errorf("%s: decode response: %w", c.provider, err)
	}
	return toSchemaChatResponse(in), nil
}

func (c *Client) buildChatRequest(messages []schema.Message, cfg llm.ChatConfig, stream bool) (chatCompletionRequest, error) {
	if len(messages) == 0 {
		return chatCompletionRequest{}, fmt.Errorf("%s: messages required", c.provider)
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return chatCompletionRequest{}, fmt.Errorf("%s: model required (use llm.WithModel)", c.provider)
	}

	reqMsgs, err := c.mapMessages(messages)
	if err != nil {
		return chatCompletionRequest{}, err
	}

	req := chatCompletionRequest{
		provider: c.provider,
		Model:    cfg.Model,
		Messages: reqMsgs,
		Stream:   stream,
	}

	req.Temperature = cfg.Temperature
	req.TopP = cfg.TopP
	req.FrequencyPenalty = cfg.FrequencyPenalty
	req.PresencePenalty = cfg.PresencePenalty
	req.Seed = cfg.Seed

	if cfg.MaxCompletionTokens != nil {
		req.MaxCompletionTokens = cfg.MaxCompletionTokens
	} else {
		req.MaxTokens = cfg.MaxTokens
	}

	if len(cfg.Tools) > 0 {
		tools, err := toWireTools(cfg.Tools)
		if err != nil {
			return chatCompletionRequest{}, err
		}
		req.Tools = tools
	}
	if cfg.ToolChoice != nil {
		req.ToolChoice = toWireToolChoice(*cfg.ToolChoice)
	}
	req.ParallelToolCalls = cfg.ParallelToolCalls

	if cfg.Stop != nil {
		req.Stop = *cfg.Stop
	}
	req.Logprobs = cfg.Logprobs
	req.TopLogprobs = cfg.TopLogprobs
	req.N = cfg.N
	if len(cfg.Metadata) > 0 {
		req.Metadata = cfg.Metadata
	}
	if len(cfg.LogitBias) > 0 {
		req.LogitBias = cfg.LogitBias
	}
	req.ServiceTier = cfg.ServiceTier
	req.User = cfg.User

	if cfg.ResponseFormat != nil {
		rf, err := toWireResponseFormat(*cfg.ResponseFormat)
		if err != nil {
			return chatCompletionRequest{}, err
		}
		req.ResponseFormat = rf
	}
	if cfg.StreamOptions != nil {
		data, err := json.Marshal(cfg.StreamOptions)
		if err != nil {
			return chatCompletionRequest{}, fmt.Errorf("%s: marshal stream_options: %w", c.provider, err)
		}
		if !bytes.Equal(data, []byte("{}")) && !bytes.Equal(data, []byte("null")) {
			req.StreamOptions = json.RawMessage(data)
		}
	}

	req.extra = cfg.ExtraFields
	req.allowExtraFieldOverride = cfg.AllowExtraFieldOverride

	return req, nil
}

func (c *Client) mapMessages(messages []schema.Message) ([]wireRequestMessage, error) {
	reqMsgs := make([]wireRequestMessage, 0, len(messages))
	for _, m := range messages {
		wm, err := toWireMessage(c.provider, m)
		if err != nil {
			return nil, err
		}
		reqMsgs = append(reqMsgs, wm)
	}
	return reqMsgs, nil
}

func (c *Client) mapChatResponseBytes(raw []byte, keepRaw bool, hooks []llm.ResponseHook) (schema.ChatResponse, error) {
	var in chatCompletionResponse
	if err := json.Unmarshal(raw, &in); err != nil {
		return schema.ChatResponse{}, fmt.Errorf("%s: decode response: %w", c.provider, err)
	}

	out := toSchemaChatResponse(in)
	if keepRaw {
		out.Raw = json.RawMessage(raw)
	}

	for _, h := range hooks {
		if h == nil {
			continue
		}
		if err := h(&out, json.RawMessage(raw)); err != nil {
			return schema.ChatResponse{}, err
		}
	}

	return out, nil
}
