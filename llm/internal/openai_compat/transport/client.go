package transport

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
	"strconv"
	"strings"
	"time"

	"github.com/lgc202/go-kit/llm"
)

const (
	httpContentTypeJSON = "application/json"
)

// 限制错误响应体最大读取 1MB
const maxErrorBodyBytes = 1 << 20

type Config struct {
	Provider llm.Provider

	BaseURL     string
	Path        string
	DefaultPath string

	APIKey     string
	HTTPClient *http.Client

	// DefaultHeaders 默认请求头，会被请求级别的 headers 覆盖
	DefaultHeaders http.Header
}

type RequestConfig struct {
	Timeout    *time.Duration
	Headers    http.Header
	ErrorHooks []llm.ErrorHook
}

type Client struct {
	provider string

	baseURL *url.URL
	path    string

	apiKey        string
	httpClient    *http.Client
	defaultHeader http.Header
}

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
	defPath := strings.TrimSpace(cfg.DefaultPath)
	basePath := strings.TrimRight(u.Path, "/")
	if defPath != "" && (path == "" || path == defPath) {
		// 用户传入完整端点 URL 时（如 ".../chat/completions"），避免重复路径
		if strings.HasSuffix(basePath, defPath) {
			path = ""
		} else if path == "" {
			path = defPath
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
	}, nil
}

func (c *Client) Provider() string { return c.provider }

func (c *Client) PostJSON(ctx context.Context, payload any, cfg RequestConfig, accept string) (*http.Response, error) {
	var cancel context.CancelFunc
	if cfg.Timeout != nil {
		ctx, cancel = context.WithTimeout(ctx, *cfg.Timeout)
		defer cancel()
	}

	body, err := json.Marshal(payload)
	if err != nil {
		return nil, fmt.Errorf("%s: marshal request: %w", c.provider, err)
	}

	req, err := http.NewRequestWithContext(ctx, http.MethodPost, c.endpoint(), bytes.NewReader(body))
	if err != nil {
		return nil, fmt.Errorf("%s: new request: %w", c.provider, err)
	}

	c.applyHeaders(req, cfg)
	if strings.TrimSpace(accept) != "" {
		req.Header.Set("Accept", accept)
	}

	resp, err := c.httpClient.Do(req)
	if err != nil {
		return nil, fmt.Errorf("%s: do request: %w", c.provider, sanitizeHTTPError(err))
	}

	if resp.StatusCode < 200 || resp.StatusCode >= 300 {
		defer resp.Body.Close()
		respBytes, rerr := readLimited(resp.Body, maxErrorBodyBytes)
		if rerr != nil {
			return nil, fmt.Errorf("%s: http %d (also failed to read error body: %v)", c.provider, resp.StatusCode, rerr)
		}
		return nil, parseError(llm.Provider(c.provider), resp.StatusCode, resp.Header, respBytes, cfg.ErrorHooks)
	}

	return resp, nil
}

func (c *Client) applyHeaders(req *http.Request, cfg RequestConfig) {
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

type errorResponse struct {
	Error struct {
		Message string          `json:"message"`
		Type    string          `json:"type"`
		Param   json.RawMessage `json:"param"`
		Code    string          `json:"code"`
	} `json:"error"`
}

func parseError(provider llm.Provider, statusCode int, hdr http.Header, body []byte, hooks []llm.ErrorHook) error {
	for _, h := range hooks {
		if h == nil {
			continue
		}
		if err := h(provider, statusCode, body); err != nil {
			return err
		}
	}

	var er errorResponse
	if err := json.Unmarshal(body, &er); err == nil && strings.TrimSpace(er.Error.Message) != "" {
		return &llm.APIError{
			Provider:   provider,
			StatusCode: statusCode,
			Code:       strings.TrimSpace(er.Error.Code),
			Type:       strings.TrimSpace(er.Error.Type),
			Message:    strings.TrimSpace(er.Error.Message),
			RequestID:  extractRequestID(hdr),
			RetryAfter: parseRetryAfter(hdr),
			Raw:        slices.Clone(body),
		}
	}

	msg := strings.TrimSpace(string(body))
	if msg == "" {
		msg = http.StatusText(statusCode)
	}

	return &llm.APIError{
		Provider:   provider,
		StatusCode: statusCode,
		Message:    msg,
		RequestID:  extractRequestID(hdr),
		RetryAfter: parseRetryAfter(hdr),
		Raw:        slices.Clone(body),
	}
}

func sanitizeHTTPError(err error) error {
	if err == nil {
		return nil
	}

	if errors.Is(err, context.DeadlineExceeded) {
		return fmt.Errorf("request timeout: API call exceeded deadline: %w", err)
	}
	if errors.Is(err, context.Canceled) {
		return fmt.Errorf("request cancelled: %w", err)
	}

	var netErr net.Error
	if errors.As(err, &netErr) {
		if netErr.Timeout() {
			return fmt.Errorf("request timeout: network operation exceeded timeout: %w", err)
		}
		return fmt.Errorf("network error: failed to reach API server: %w", err)
	}

	return err
}

func readLimited(r io.Reader, limit int) ([]byte, error) {
	lr := &io.LimitedReader{R: r, N: int64(limit)}
	b, err := io.ReadAll(lr)
	if err != nil {
		return nil, err
	}
	return b, nil
}

func extractRequestID(h http.Header) string {
	if h == nil {
		return ""
	}

	for _, k := range []string{
		"X-Request-Id",
		"X-Request-ID",
		"X-RequestId",
		"X-Amzn-RequestId",
	} {
		if v := strings.TrimSpace(h.Get(k)); v != "" {
			return v
		}
	}
	return ""
}

func parseRetryAfter(h http.Header) time.Duration {
	if h == nil {
		return 0
	}
	v := strings.TrimSpace(h.Get("Retry-After"))
	if v == "" {
		return 0
	}
	// RFC 9110 支持秒数或 HTTP-date 格式
	if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second
	}
	if t, err := http.ParseTime(v); err == nil {
		d := time.Until(t)
		if d < 0 {
			return 0
		}
		return d
	}
	return 0
}
