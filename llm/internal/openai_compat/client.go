package openai_compat

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"net"
	"net/http"
	"net/url"
	"slices"
	"strings"

	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/schema"
)

const (
	httpContentTypeJSON = "application/json"
	httpAcceptJSON      = "application/json"
	httpAcceptSSE       = "text/event-stream"
)

const maxErrorBodyBytes = 1 << 20

type Config struct {
	Provider llm.Provider

	BaseURL string
	Path    string

	APIKey     string
	HTTPClient *http.Client

	// DefaultHeaders 首先应用，然后被请求级别的 headers 覆盖
	DefaultHeaders http.Header

	// DefaultOptions 提供客户端级别的默认请求选项
	DefaultOptions []llm.RequestOption

	Adapter Adapter
}

type Client struct {
	provider string

	baseURL *url.URL
	path    string

	apiKey        string
	httpClient    *http.Client
	defaultHeader http.Header
	defaultOpts   []llm.RequestOption
	adapter       Adapter
}

var _ llm.ChatModel = (*Client)(nil)

func New(cfg Config) (*Client, error) {
	if strings.TrimSpace(string(cfg.Provider)) == "" {
		return nil, fmt.Errorf("openai_compat: provider required")
	}

	base := strings.TrimSpace(cfg.BaseURL)
	if base == "" {
		return nil, fmt.Errorf("openai_compat: base url required")
	}
	u, err := url.Parse(base)
	if err != nil {
		return nil, fmt.Errorf("openai_compat: parse base url: %w", err)
	}

	path := strings.TrimSpace(cfg.Path)
	basePath := strings.TrimRight(u.Path, "/")
	if path == "" || path == defaultChatCompletionsPath {
		// If user passes a full endpoint URL like ".../chat/completions" (common copy/paste),
		// avoid duplicating it when Path is omitted (or left as default).
		if strings.HasSuffix(basePath, defaultChatCompletionsPath) {
			path = ""
		} else if path == "" {
			path = defaultChatCompletionsPath
		}
	}
	if path != "" && !strings.HasPrefix(path, "/") {
		path = "/" + path
	}

	hc := cfg.HTTPClient
	if hc == nil {
		hc = http.DefaultClient
	}

	var hdr http.Header
	if cfg.DefaultHeaders != nil {
		hdr = cfg.DefaultHeaders.Clone()
	}

	return &Client{
		provider:      string(cfg.Provider),
		baseURL:       u,
		path:          path,
		apiKey:        cfg.APIKey,
		httpClient:    hc,
		defaultHeader: hdr,
		defaultOpts:   slices.Clone(cfg.DefaultOptions),
		adapter:       cfg.Adapter,
	}, nil
}

func (c *Client) Chat(ctx context.Context, messages []schema.Message, opts ...llm.RequestOption) (schema.ChatResponse, error) {
	reqCfg := llm.ApplyRequestOptions(slices.Concat(c.defaultOpts, opts)...)
	var cancel context.CancelFunc
	if reqCfg.Timeout != nil {
		ctx, cancel = context.WithTimeout(ctx, *reqCfg.Timeout)
		defer cancel()
	}

	payload, err := c.buildChatRequest(messages, reqCfg, false)
	if err != nil {
		return schema.ChatResponse{}, err
	}

	resp, err := c.doRequest(ctx, payload, reqCfg, httpAcceptJSON)
	if err != nil {
		return schema.ChatResponse{}, err
	}
	defer resp.Body.Close()

	return c.parseChatResponse(resp, reqCfg)
}

func (c *Client) ChatStream(ctx context.Context, messages []schema.Message, opts ...llm.RequestOption) (llm.Stream, error) {
	reqCfg := llm.ApplyRequestOptions(slices.Concat(c.defaultOpts, opts)...)
	var cancel context.CancelFunc
	if reqCfg.Timeout != nil {
		ctx, cancel = context.WithTimeout(ctx, *reqCfg.Timeout)
		defer cancel()
	}

	payload, err := c.buildChatRequest(messages, reqCfg, true)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, payload, reqCfg, httpAcceptSSE)
	if err != nil {
		return nil, err
	}

	return newStream(c.provider, c.adapter, resp.Body, reqCfg.KeepRaw), nil
}

// doRequest 执行 HTTP 请求
func (c *Client) doRequest(ctx context.Context, payload any, cfg llm.RequestConfig, accept string) (*http.Response, error) {
	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%s: marshal request: %w", c.provider, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%s: new request: %w", c.provider, err)
	}

	c.applyHeaders(req, cfg)
	req.Header.Set("Accept", accept)

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: do request: %w", c.provider, sanitizeHTTPError(err))
	}

	// 检查 HTTP 状态码
	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		respBytes, rerr := readLimited(resp.Body, maxErrorBodyBytes)
		if rerr != nil {
			return nil, fmt.Errorf("%s: http %d (also failed to read error body: %v)", c.provider, resp.StatusCode, rerr)
		}
		return nil, c.parseError(resp.StatusCode, respBytes)
	}

	return resp, nil
}

// parseChatResponse 解析 chat 响应
func (c *Client) parseChatResponse(resp *http.Response, cfg llm.RequestConfig) (schema.ChatResponse, error) {
	if cfg.KeepRaw || c.adapter != nil {
		respBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return schema.ChatResponse{}, fmt.Errorf("%s: read response: %w", c.provider, err)
		}
		return c.mapChatResponseBytes(respBytes, cfg.KeepRaw)
	}

	var in chatCompletionResponse
	if err := json.NewDecoder(resp.Body).Decode(&in); err != nil {
		return schema.ChatResponse{}, fmt.Errorf("%s: decode response: %w", c.provider, err)
	}
	return toSchemaChatResponse(in), nil
}

func (c *Client) buildChatRequest(messages []schema.Message, cfg llm.RequestConfig, stream bool) (chatCompletionRequest, error) {
	// 验证必需参数
	if len(messages) == 0 {
		return chatCompletionRequest{}, fmt.Errorf("%s: messages required", c.provider)
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return chatCompletionRequest{}, fmt.Errorf("%s: model required (use llm.WithModel)", c.provider)
	}

	// 构建消息列表
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

	// sampling
	req.Temperature = cfg.Temperature
	req.TopP = cfg.TopP
	req.FrequencyPenalty = cfg.FrequencyPenalty
	req.PresencePenalty = cfg.PresencePenalty
	req.Seed = cfg.Seed

	// tokens
	if cfg.MaxCompletionTokens != nil {
		req.MaxCompletionTokens = cfg.MaxCompletionTokens
	} else {
		req.MaxTokens = cfg.MaxTokens
	}

	// tools
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

	// optional
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

	// ExtraFields merged at marshal-time (with conflict checks).
	req.extra = cfg.ExtraFields
	req.allowExtraFieldOverride = cfg.AllowExtraFieldOverride

	// provider-specific extensions
	if c.adapter != nil {
		if err := c.adapter.ApplyRequestExtensions(&req, cfg); err != nil {
			return chatCompletionRequest{}, err
		}
	}

	return req, nil
}

// mapMessages 将 schema.Message 列表转换为请求格式
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

func (c *Client) mapChatResponseBytes(raw []byte, keepRaw bool) (schema.ChatResponse, error) {
	var in chatCompletionResponse
	if err := json.Unmarshal(raw, &in); err != nil {
		return schema.ChatResponse{}, fmt.Errorf("%s: decode response: %w", c.provider, err)
	}

	out := toSchemaChatResponse(in)
	if keepRaw {
		out.Raw = json.RawMessage(raw)
	}

	// 调用 adapter 丰富响应数据
	if c.adapter != nil {
		if err := c.adapter.EnrichResponse(&out, json.RawMessage(raw)); err != nil {
			return schema.ChatResponse{}, err
		}
	}

	return out, nil
}

// parseError 解析 API 错误响应
func (c *Client) parseError(statusCode int, body []byte) error {
	// 首先尝试使用 adapter 解析 provider 特定错误
	if c.adapter != nil {
		if err := c.adapter.ParseError(c.provider, statusCode, body); err != nil {
			return err
		}
	}

	// 尝试解析标准 OpenAI 兼容错误格式
	var er errorResponse
	if err := json.Unmarshal(body, &er); err == nil && er.Error.Message != "" {
		if er.Error.Code != "" {
			return fmt.Errorf("%s: http %d: %s (%s)", c.provider, statusCode, er.Error.Message, er.Error.Code)
		}
		return fmt.Errorf("%s: http %d: %s", c.provider, statusCode, er.Error.Message)
	}

	// 无法解析 JSON，返回原始响应体
	if len(body) > 0 {
		return fmt.Errorf("%s: http %d: %s", c.provider, statusCode, string(body))
	}
	return fmt.Errorf("%s: http %d", c.provider, statusCode)
}

// sanitizeHTTPError 清理 HTTP 客户端错误，防止泄露敏感信息
// 检查 context deadline/cancellation 错误，返回通用超时消息
// 而不是暴露请求详情、header 或其他敏感数据
func sanitizeHTTPError(err error) error {
	if err == nil {
		return nil
	}

	// 检查 context 超时
	if errors.Is(err, context.DeadlineExceeded) {
		return errors.New("request timeout: API call exceeded deadline")
	}

	// 检查 context 取消
	if errors.Is(err, context.Canceled) {
		return errors.New("request cancelled")
	}

	// 检查网络超时错误
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return errors.New("request timeout: network operation exceeded timeout")
	}

	// 对于其他网络错误，提供通用消息而不暴露细节
	if _, ok := err.(net.Error); ok {
		return errors.New("network error: failed to reach API server")
	}

	// 如果不是敏感类型，返回原始错误
	return err
}

func (c *Client) applyHeaders(req *http.Request, cfg llm.RequestConfig) {
	h := make(http.Header)
	h.Set("Content-Type", httpContentTypeJSON)

	if c.defaultHeader != nil {
		for k, vs := range c.defaultHeader {
			h[k] = slices.Clone(vs)
		}
	}
	if cfg.Headers != nil {
		for k, vs := range cfg.Headers {
			h[k] = slices.Clone(vs)
		}
	}

	// 默认使用 apiKey，但允许 DefaultHeaders / request-level headers 自行覆盖 Authorization。
	if c.apiKey != "" && h.Get("Authorization") == "" {
		h.Set("Authorization", "Bearer "+c.apiKey)
	}

	req.Header = h
}

func (c *Client) endpoint() string {
	if strings.TrimSpace(c.path) == "" {
		return c.baseURL.String()
	}
	return c.baseURL.JoinPath(strings.TrimPrefix(c.path, "/")).String()
}

func readLimited(r io.Reader, maxBytes int64) ([]byte, error) {
	lr := io.LimitReader(r, maxBytes+1)
	b, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	if int64(len(b)) > maxBytes {
		return b[:maxBytes], nil
	}
	return b, nil
}
