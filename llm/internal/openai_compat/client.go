package openai_compat

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"errors"
	"fmt"
	"io"
	"maps"
	"net"
	"net/http"
	"net/url"
	"strings"
	"time"

	"github.com/lgc202/go-kit/llm"
	"github.com/lgc202/go-kit/llm/schema"
)

type Config struct {
	Provider llm.Provider

	BaseURL string
	Path    string

	APIKey     string
	HTTPClient *http.Client

	// DefaultHeaders 首先应用，然后被请求级别的 headers 覆盖
	DefaultHeaders http.Header

	// DefaultRequest 提供客户端级别的默认请求选项
	DefaultRequest llm.RequestConfig

	Adapter Adapter
}

type Client struct {
	provider string

	baseURL *url.URL
	path    string

	apiKey        string
	httpClient    *http.Client
	defaultHeader http.Header
	defaultReq    llm.RequestConfig
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
	if path == "" {
		path = "/chat/completions"
	}
	if !strings.HasPrefix(path, "/") {
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

	ad := cfg.Adapter
	if ad == nil {
		ad = NoopAdapter{}
	}

	return &Client{
		provider:      string(cfg.Provider),
		baseURL:       u,
		path:          path,
		apiKey:        cfg.APIKey,
		httpClient:    hc,
		defaultHeader: hdr,
		defaultReq:    cfg.DefaultRequest,
		adapter:       ad,
	}, nil
}

func (c *Client) Chat(ctx context.Context, messages []schema.Message, opts ...llm.RequestOption) (schema.ChatResponse, error) {
	reqCfg := llm.ApplyRequestOptions(c.defaultReq, opts...)
	var cancel context.CancelFunc
	if reqCfg.Timeout != nil {
		ctx, cancel = context.WithTimeout(ctx, *reqCfg.Timeout)
		defer cancel()
	}

	payload, err := c.buildChatRequest(messages, reqCfg, false)
	if err != nil {
		return schema.ChatResponse{}, err
	}

	resp, err := c.doRequest(ctx, payload, reqCfg, "application/json")
	if err != nil {
		return schema.ChatResponse{}, err
	}
	defer resp.Body.Close()

	return c.parseChatResponse(resp, reqCfg)
}

func (c *Client) ChatStream(ctx context.Context, messages []schema.Message, opts ...llm.RequestOption) (llm.Stream, error) {
	reqCfg := llm.ApplyRequestOptions(c.defaultReq, opts...)
	var cancel context.CancelFunc
	if reqCfg.Timeout != nil {
		ctx, cancel = context.WithTimeout(ctx, *reqCfg.Timeout)
		defer cancel()
	}

	payload, err := c.buildChatRequest(messages, reqCfg, true)
	if err != nil {
		return nil, err
	}

	resp, err := c.doRequest(ctx, payload, reqCfg, "text/event-stream")
	if err != nil {
		return nil, err
	}

	return newStream(c.provider, c.adapter, resp.Body, reqCfg.KeepRaw), nil
}

// doRequest 执行 HTTP 请求
func (c *Client) doRequest(ctx context.Context, payload map[string]any, cfg llm.RequestConfig, accept string) (*http.Response, error) {
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
		respBytes, rerr := io.ReadAll(resp.Body)
		if rerr != nil {
			return nil, fmt.Errorf("%s: http %d (also failed to read error body: %v)", c.provider, resp.StatusCode, rerr)
		}
		return nil, c.parseError(resp.StatusCode, respBytes)
	}

	return resp, nil
}

// parseChatResponse 解析 chat 响应
func (c *Client) parseChatResponse(resp *http.Response, cfg llm.RequestConfig) (schema.ChatResponse, error) {
	respBytes, err := io.ReadAll(resp.Body)
	if err != nil {
		return schema.ChatResponse{}, fmt.Errorf("%s: read response: %w", c.provider, err)
	}

	return c.mapChatResponse(respBytes, cfg.KeepRaw)
}

func (c *Client) buildChatRequest(messages []schema.Message, cfg llm.RequestConfig, stream bool) (map[string]any, error) {
	// 验证必需参数
	if len(messages) == 0 {
		return nil, fmt.Errorf("%s: messages required", c.provider)
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return nil, fmt.Errorf("%s: model required (use llm.WithModel)", c.provider)
	}

	// 构建消息列表
	reqMsgs, err := c.mapMessages(messages)
	if err != nil {
		return nil, err
	}

	// 构建基础请求
	req := map[string]any{
		"model":    cfg.Model,
		"messages": reqMsgs,
		"stream":   stream,
	}

	// 应用采样参数
	c.applySamplingParams(req, cfg)

	// 应用 token 限制参数
	c.applyTokenParams(req, cfg)

	// 应用工具调用参数
	if err := c.applyToolParams(req, cfg); err != nil {
		return nil, err
	}

	// 应用其他可选参数
	c.applyOptionalParams(req, cfg)

	// 应用 provider 特定扩展
	if err := c.applyProviderExtensions(req, cfg); err != nil {
		return nil, err
	}

	return req, nil
}

// applySamplingParams 应用采样相关参数（温度、top_p、惩罚等）
func (c *Client) applySamplingParams(req map[string]any, cfg llm.RequestConfig) {
	if cfg.Temperature != nil {
		req["temperature"] = *cfg.Temperature
	}
	if cfg.TopP != nil {
		req["top_p"] = *cfg.TopP
	}
	if cfg.FrequencyPenalty != nil {
		req["frequency_penalty"] = *cfg.FrequencyPenalty
	}
	if cfg.PresencePenalty != nil {
		req["presence_penalty"] = *cfg.PresencePenalty
	}
	if cfg.Seed != nil {
		req["seed"] = *cfg.Seed
	}
}

// applyTokenParams 应用 token 限制参数
func (c *Client) applyTokenParams(req map[string]any, cfg llm.RequestConfig) {
	// 优先使用 max_completion_tokens，如果没有则使用 max_tokens
	if cfg.MaxCompletionTokens != nil {
		req["max_completion_tokens"] = *cfg.MaxCompletionTokens
	} else if cfg.MaxTokens != nil {
		req["max_tokens"] = *cfg.MaxTokens
	}
}

// applyToolParams 应用工具调用相关参数
func (c *Client) applyToolParams(req map[string]any, cfg llm.RequestConfig) error {
	if len(cfg.Tools) > 0 {
		tools, err := mapTools(cfg.Tools)
		if err != nil {
			return err
		}
		req["tools"] = tools
	}
	if cfg.ToolChoice != nil {
		req["tool_choice"] = mapToolChoice(*cfg.ToolChoice)
	}
	if cfg.ParallelToolCalls != nil {
		req["parallel_tool_calls"] = *cfg.ParallelToolCalls
	}
	return nil
}

// applyOptionalParams 应用其他可选参数
func (c *Client) applyOptionalParams(req map[string]any, cfg llm.RequestConfig) error {
	if cfg.Stop != nil {
		req["stop"] = *cfg.Stop
	}
	if cfg.Logprobs != nil {
		req["logprobs"] = *cfg.Logprobs
	}
	if cfg.TopLogprobs != nil {
		req["top_logprobs"] = *cfg.TopLogprobs
	}
	if cfg.N != nil {
		req["n"] = *cfg.N
	}
	if cfg.Metadata != nil && len(cfg.Metadata) > 0 {
		req["metadata"] = cfg.Metadata
	}
	if cfg.LogitBias != nil && len(cfg.LogitBias) > 0 {
		req["logit_bias"] = cfg.LogitBias
	}
	if cfg.ServiceTier != nil {
		req["service_tier"] = *cfg.ServiceTier
	}
	if cfg.User != nil {
		req["user"] = *cfg.User
	}
	if cfg.ResponseFormat != nil {
		rf, err := mapResponseFormat(*cfg.ResponseFormat)
		if err != nil {
			return err
		}
		req["response_format"] = rf
	}
	if cfg.StreamOptions != nil {
		data, err := json.Marshal(cfg.StreamOptions)
		if err != nil {
			return fmt.Errorf("%s: marshal stream_options: %w", c.provider, err)
		}
		// 只在非零值时才添加，避免发送 null
		if !bytes.Equal(data, []byte("{}")) && !bytes.Equal(data, []byte("null")) {
			req["stream_options"] = json.RawMessage(data)
		}
	}
	if cfg.ExtraFields != nil {
		maps.Copy(req, cfg.ExtraFields)
	}
	return nil
}

// applyProviderExtensions 应用 provider 特定的扩展
func (c *Client) applyProviderExtensions(req map[string]any, cfg llm.RequestConfig) error {
	if c.adapter != nil {
		return c.adapter.ApplyRequestExtensions(req, cfg)
	}
	return nil
}

// mapMessages 将 schema.Message 列表转换为请求格式
func (c *Client) mapMessages(messages []schema.Message) ([]map[string]any, error) {
	reqMsgs := make([]map[string]any, 0, len(messages))
	for _, m := range messages {
		wm, err := c.mapRequestMessage(m)
		if err != nil {
			return nil, err
		}
		reqMsgs = append(reqMsgs, wm)
	}
	return reqMsgs, nil
}

func (c *Client) mapRequestMessage(m schema.Message) (map[string]any, error) {
	out := map[string]any{
		"role": string(m.Role),
	}
	if m.Name != "" {
		out["name"] = m.Name
	}
	if m.ToolCallID != "" {
		out["tool_call_id"] = m.ToolCallID
	}

	if len(m.Content) > 0 {
		if len(m.Content) == 1 {
			if tp, ok := m.Content[0].(schema.TextContent); ok {
				out["content"] = tp.Text
				return out, nil
			}
		}

		parts := make([]map[string]any, 0, len(m.Content))
		for _, p := range m.Content {
			switch part := p.(type) {
			case schema.TextContent:
				parts = append(parts, map[string]any{
					"type": "text",
					"text": part.Text,
				})
			case schema.ImageURLContent:
				imageURL := map[string]any{
					"url": part.URL,
				}
				if strings.TrimSpace(part.Detail) != "" {
					imageURL["detail"] = part.Detail
				}
				parts = append(parts, map[string]any{
					"type":      "image_url",
					"image_url": imageURL,
				})
			case schema.BinaryContent:
				if strings.TrimSpace(part.MIMEType) == "" {
					return nil, fmt.Errorf("%s: binary mime type required", c.provider)
				}
				if len(part.Data) == 0 {
					return nil, fmt.Errorf("%s: binary data required", c.provider)
				}
				dataURL := "data:" + part.MIMEType + ";base64," + base64.StdEncoding.EncodeToString(part.Data)
				parts = append(parts, map[string]any{
					"type": "image_url",
					"image_url": map[string]any{
						"url": dataURL,
					},
				})
			default:
				return nil, &llm.UnsupportedOptionError{
					Provider: llm.Provider(c.provider),
					Option:   "message.content",
					Reason:   fmt.Sprintf("unsupported content part type %T", p),
				}
			}
		}
		out["content"] = parts
		return out, nil
	}

	out["content"] = ""
	return out, nil
}

func mapTools(tools []schema.Tool) ([]map[string]any, error) {
	out := make([]map[string]any, 0, len(tools))
	for _, t := range tools {
		if t.Type != schema.ToolTypeFunction {
			continue
		}
		fn := map[string]any{
			"name": t.Function.Name,
		}
		if t.Function.Description != "" {
			fn["description"] = t.Function.Description
		}
		if len(t.Function.Parameters) > 0 {
			if !json.Valid(t.Function.Parameters) {
				return nil, fmt.Errorf("openai_compat: invalid tool parameters JSON for %q", t.Function.Name)
			}
			fn["parameters"] = json.RawMessage(t.Function.Parameters)
		}
		if t.Function.Strict {
			fn["strict"] = true
		}
		out = append(out, map[string]any{
			"type":     "function",
			"function": fn,
		})
	}
	return out, nil
}

func mapToolChoice(tc schema.ToolChoice) any {
	switch tc.Mode {
	case schema.ToolChoiceNone:
		return "none"
	case schema.ToolChoiceAuto:
		return "auto"
	default:
		if tc.FunctionName != "" {
			return map[string]any{
				"type": "function",
				"function": map[string]any{
					"name": tc.FunctionName,
				},
			}
		}
		return "auto"
	}
}

func mapResponseFormat(rf schema.ResponseFormat) (map[string]any, error) {
	out := map[string]any{
		"type": rf.Type,
	}
	if len(rf.JSONSchema) > 0 {
		if !json.Valid(rf.JSONSchema) {
			return nil, fmt.Errorf("openai_compat: invalid response_format.json_schema JSON")
		}
		out["json_schema"] = json.RawMessage(rf.JSONSchema)
	}
	return out, nil
}

func (c *Client) mapChatResponse(raw []byte, keepRaw bool) (schema.ChatResponse, error) {
	var in chatCompletionResponse
	if err := json.Unmarshal(raw, &in); err != nil {
		return schema.ChatResponse{}, fmt.Errorf("%s: decode response: %w", c.provider, err)
	}

	out := schema.ChatResponse{
		ID:    in.ID,
		Model: in.Model,
		Usage: schema.Usage{
			PromptTokens:          in.Usage.PromptTokens,
			CompletionTokens:      in.Usage.CompletionTokens,
			TotalTokens:           in.Usage.TotalTokens,
			PromptCacheHitTokens:  in.Usage.PromptCacheHitTokens,
			PromptCacheMissTokens: in.Usage.PromptCacheMissTokens,
			CachedTokens:          in.Usage.CachedTokens,
		},
		ServiceTier: in.ServiceTier,
	}
	if keepRaw {
		out.Raw = json.RawMessage(raw)
	}
	if in.Usage.CompletionTokensDetails != nil && in.Usage.CompletionTokensDetails.ReasoningTokens != 0 {
		out.Usage.CompletionTokensDetails = &schema.CompletionTokensDetails{
			ReasoningTokens: in.Usage.CompletionTokensDetails.ReasoningTokens,
		}
	}
	if in.Created != 0 {
		out.CreatedAt = time.Unix(in.Created, 0)
	}

	out.Choices = make([]schema.Choice, 0, len(in.Choices))
	for _, c0 := range in.Choices {
		out.Choices = append(out.Choices, schema.Choice{
			Index:        c0.Index,
			Message:      mapWireMessage(c0.Message),
			FinishReason: schema.FinishReason(c0.FinishReason),
		})
	}

	// 调用 adapter 丰富响应数据
	if c.adapter != nil {
		if err := c.adapter.EnrichResponse(&out, json.RawMessage(raw)); err != nil {
			return schema.ChatResponse{}, err
		}
	}

	return out, nil
}

func mapWireMessage(m wireMessage) schema.Message {
	parts := normalizeWireContent(m.Content)
	out := schema.Message{
		Role:             schema.Role(m.Role),
		Content:          parts,
		Name:             m.Name,
		ToolCallID:       m.ToolCallID,
		ReasoningContent: m.ReasoningContent,
	}

	if len(m.ToolCalls) > 0 {
		out.ToolCalls = make([]schema.ToolCall, 0, len(m.ToolCalls))
		for _, tc := range m.ToolCalls {
			out.ToolCalls = append(out.ToolCalls, schema.ToolCall{
				ID:   tc.ID,
				Type: schema.ToolCallType(tc.Type),
				Function: schema.ToolFunction{
					Name:      tc.Function.Name,
					Arguments: tc.Function.Arguments,
				},
			})
		}
	}

	return out
}

type wireContentPart struct {
	Type string `json:"type"`

	Text string `json:"text,omitempty"`

	ImageURL *struct {
		URL    string `json:"url"`
		Detail string `json:"detail,omitempty"`
	} `json:"image_url,omitempty"`
}

func normalizeWireContent(in any) []schema.ContentPart {
	switch v := in.(type) {
	case string:
		if v == "" {
			return nil
		}
		return []schema.ContentPart{schema.TextContent{Text: v}}
	case []any:
		parts := make([]schema.ContentPart, 0, len(v))
		for _, p := range v {
			b, err := json.Marshal(p)
			if err != nil {
				continue
			}
			var wp wireContentPart
			if err := json.Unmarshal(b, &wp); err != nil {
				continue
			}
			switch wp.Type {
			case "text":
				parts = append(parts, schema.TextContent{Text: wp.Text})
			case "image_url":
				if wp.ImageURL != nil {
					parts = append(parts, schema.ImageURLContent{URL: wp.ImageURL.URL, Detail: wp.ImageURL.Detail})
				}
			}
		}
		return parts
	default:
		return nil
	}
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
		return &llm.APIError{
			Provider:   c.provider,
			StatusCode: statusCode,
			Message:    er.Error.Message,
			Type:       er.Error.Type,
			Code:       er.Error.Code,
		}
	}

	// 无法解析 JSON，返回原始响应体
	return &llm.APIError{
		Provider:   c.provider,
		StatusCode: statusCode,
		Body:       body,
	}
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
		return fmt.Errorf("request timeout: API call exceeded deadline")
	}

	// 检查 context 取消
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("request cancelled")
	}

	// 检查网络超时错误
	var netErr net.Error
	if errors.As(err, &netErr) && netErr.Timeout() {
		return fmt.Errorf("request timeout: network operation exceeded timeout")
	}

	// 对于其他网络错误，提供通用消息而不暴露细节
	if _, ok := err.(net.Error); ok {
		return fmt.Errorf("network error: failed to reach API server")
	}

	// 如果不是敏感类型，返回原始错误
	return err
}

func (c *Client) applyHeaders(req *http.Request, cfg llm.RequestConfig) {
	req.Header.Set("Content-Type", "application/json")

	if c.apiKey != "" && req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+c.apiKey)
	}

	if c.defaultHeader != nil {
		for k, vs := range c.defaultHeader {
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
	}
	if cfg.Headers != nil {
		for k, vs := range cfg.Headers {
			req.Header.Del(k)
			for _, v := range vs {
				req.Header.Add(k, v)
			}
		}
	}
}

func (c *Client) endpoint() string {
	u := *c.baseURL
	u.Path = strings.TrimRight(u.Path, "/") + c.path
	return u.String()
}
