package openai_compat

import (
	"bytes"
	"context"
	"encoding/base64"
	"encoding/json"
	"fmt"
	"io"
	"maps"
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

	// DefaultHeaders are applied first, then overridden by request-level headers.
	DefaultHeaders http.Header

	// DefaultRequest provides client-level defaults for request options.
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
	if reqCfg.Timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *reqCfg.Timeout)
		defer cancel()
	}

	payload, err := c.buildChatRequest(messages, reqCfg, false)
	if err != nil {
		return schema.ChatResponse{}, err
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return schema.ChatResponse{}, fmt.Errorf("%s: marshal request: %w", c.provider, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(), bytes.NewReader(body))
	if err != nil {
		return schema.ChatResponse{}, fmt.Errorf("%s: new request: %w", c.provider, err)
	}
	c.applyHeaders(req, reqCfg)
	req.Header.Set("Accept", "application/json")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return schema.ChatResponse{}, fmt.Errorf("%s: do request: %w", c.provider, err)
	}
	defer resp.Body.Close()

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		respBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return schema.ChatResponse{}, fmt.Errorf("%s: read response: %w", c.provider, err)
		}
		return schema.ChatResponse{}, c.parseError(resp.StatusCode, respBytes)
	}

	var out chatCompletionResponse
	var raw []byte
	if reqCfg.KeepRaw {
		respBytes, err := io.ReadAll(resp.Body)
		if err != nil {
			return schema.ChatResponse{}, fmt.Errorf("%s: read response: %w", c.provider, err)
		}
		raw = respBytes
		if err := json.Unmarshal(respBytes, &out); err != nil {
			return schema.ChatResponse{}, fmt.Errorf("%s: decode response: %w", c.provider, err)
		}
	} else {
		if err := json.NewDecoder(resp.Body).Decode(&out); err != nil {
			return schema.ChatResponse{}, fmt.Errorf("%s: decode response: %w", c.provider, err)
		}
	}

	return c.mapChatResponse(out, raw, reqCfg.KeepRaw)
}

func (c *Client) ChatStream(ctx context.Context, messages []schema.Message, opts ...llm.RequestOption) (llm.Stream, error) {
	reqCfg := llm.ApplyRequestOptions(c.defaultReq, opts...)
	if reqCfg.Timeout != nil {
		var cancel context.CancelFunc
		ctx, cancel = context.WithTimeout(ctx, *reqCfg.Timeout)
		defer cancel()
	}

	payload, err := c.buildChatRequest(messages, reqCfg, true)
	if err != nil {
		return nil, err
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%s: marshal request: %w", c.provider, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%s: new request: %w", c.provider, err)
	}
	c.applyHeaders(req, reqCfg)
	req.Header.Set("Accept", "text/event-stream")

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: do request: %w", c.provider, err)
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		respBytes, readErr := io.ReadAll(resp.Body)
		if readErr != nil {
			return nil, fmt.Errorf("%s: http %d (also failed to read error body: %v)", c.provider, resp.StatusCode, readErr)
		}
		return nil, c.parseError(resp.StatusCode, respBytes)
	}

	return newStream(c.provider, c.adapter, resp.Body, reqCfg.KeepRaw), nil
}

func (c *Client) buildChatRequest(messages []schema.Message, cfg llm.RequestConfig, stream bool) (map[string]any, error) {
	if len(messages) == 0 {
		return nil, fmt.Errorf("%s: messages required", c.provider)
	}
	if strings.TrimSpace(cfg.Model) == "" {
		return nil, fmt.Errorf("%s: model required (use llm.WithModel)", c.provider)
	}

	reqMsgs := make([]map[string]any, 0, len(messages))
	for _, m := range messages {
		wm, err := c.mapRequestMessage(m)
		if err != nil {
			return nil, err
		}
		reqMsgs = append(reqMsgs, wm)
	}

	req := map[string]any{
		"model":    cfg.Model,
		"messages": reqMsgs,
		"stream":   stream,
	}

	if cfg.Temperature != nil {
		req["temperature"] = *cfg.Temperature
	}
	if cfg.TopP != nil {
		req["top_p"] = *cfg.TopP
	}
	// 优先使用 max_completion_tokens，如果没有则使用 max_tokens
	if cfg.MaxCompletionTokens != nil {
		req["max_completion_tokens"] = *cfg.MaxCompletionTokens
	} else if cfg.MaxTokens != nil {
		req["max_tokens"] = *cfg.MaxTokens
	}
	if cfg.Stop != nil {
		req["stop"] = *cfg.Stop
	}
	if cfg.FrequencyPenalty != nil {
		req["frequency_penalty"] = *cfg.FrequencyPenalty
	}
	if cfg.PresencePenalty != nil {
		req["presence_penalty"] = *cfg.PresencePenalty
	}
	if cfg.Logprobs != nil {
		req["logprobs"] = *cfg.Logprobs
	}
	if cfg.TopLogprobs != nil {
		req["top_logprobs"] = *cfg.TopLogprobs
	}
	if len(cfg.Tools) > 0 {
		tools, err := mapTools(cfg.Tools)
		if err != nil {
			return nil, err
		}
		req["tools"] = tools
	}
	if cfg.ToolChoice != nil {
		req["tool_choice"] = mapToolChoice(*cfg.ToolChoice)
	}
	if cfg.ParallelToolCalls != nil {
		req["parallel_tool_calls"] = *cfg.ParallelToolCalls
	}
	if cfg.ResponseFormat != nil {
		rf, err := mapResponseFormat(*cfg.ResponseFormat)
		if err != nil {
			return nil, err
		}
		req["response_format"] = rf
	}
	if cfg.N != nil {
		req["n"] = *cfg.N
	}
	if cfg.Seed != nil {
		req["seed"] = *cfg.Seed
	}
	if cfg.Metadata != nil && len(cfg.Metadata) > 0 {
		req["metadata"] = cfg.Metadata
	}
	if cfg.StreamOptions != nil {
		data, err := json.Marshal(cfg.StreamOptions)
		if err != nil {
			return nil, fmt.Errorf("%s: marshal stream_options: %w", c.provider, err)
		}
		// 只在非零值时才添加，避免发送 null
		if !bytes.Equal(data, []byte("{}")) && !bytes.Equal(data, []byte("null")) {
			req["stream_options"] = json.RawMessage(data)
		}
	}

	if cfg.ExtraFields != nil {
		maps.Copy(req, cfg.ExtraFields)
	}

	if c.adapter != nil {
		if err := c.adapter.ApplyRequestExtensions(req, cfg); err != nil {
			return nil, err
		}
	}

	return req, nil
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

func (c *Client) mapChatResponse(in chatCompletionResponse, raw []byte, keepRaw bool) (schema.ChatResponse, error) {
	out := schema.ChatResponse{
		ID:    in.ID,
		Model: in.Model,
		Usage: schema.Usage{
			PromptTokens:          in.Usage.PromptTokens,
			CompletionTokens:      in.Usage.CompletionTokens,
			TotalTokens:           in.Usage.TotalTokens,
			PromptCacheHitTokens:  in.Usage.PromptCacheHitTokens,
			PromptCacheMissTokens: in.Usage.PromptCacheMissTokens,
		},
	}
	if keepRaw && len(raw) > 0 {
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
		msg := mapWireMessage(c0.Message)
		if c.adapter != nil {
			if err := c.adapter.EnrichResponseMessage(&msg, nil); err != nil {
				return schema.ChatResponse{}, err
			}
		}

		out.Choices = append(out.Choices, schema.Choice{
			Index:        c0.Index,
			Message:      msg,
			FinishReason: schema.FinishReason(c0.FinishReason),
		})
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

func (c *Client) parseError(statusCode int, body []byte) error {
	if c.adapter != nil {
		if err := c.adapter.ParseError(c.provider, statusCode, body); err != nil {
			return err
		}
	}

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
	return &llm.APIError{
		Provider:   c.provider,
		StatusCode: statusCode,
		Body:       body,
	}
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
