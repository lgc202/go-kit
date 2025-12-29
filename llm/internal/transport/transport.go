package transport

import (
	"bytes"
	"context"
	"crypto/rand"
	"encoding/hex"
	"encoding/json"
	"errors"
	"io"
	"log/slog"
	"math"
	"math/big"
	"net/http"
	"net/url"
	"time"
)

type RetryConfig struct {
	MaxAttempts    int
	InitialBackoff time.Duration
	MaxBackoff     time.Duration
}

func DefaultRetry() RetryConfig {
	return RetryConfig{
		MaxAttempts:    3,
		InitialBackoff: 250 * time.Millisecond,
		MaxBackoff:     2 * time.Second,
	}
}

type Client struct {
	HTTPClient *http.Client
	BaseURL    *url.URL

	DefaultHeaders http.Header
	UserAgent      string
	Logger         *slog.Logger
	Retry          RetryConfig
}

func New(baseURL string, httpClient *http.Client) (*Client, error) {
	u, err := url.Parse(baseURL)
	if err != nil {
		return nil, err
	}
	if httpClient == nil {
		httpClient = &http.Client{Timeout: 60 * time.Second}
	}
	return &Client{
		HTTPClient:     httpClient,
		BaseURL:        u,
		DefaultHeaders: make(http.Header),
		UserAgent:      "go-kit-llm/1",
		Logger:         slog.New(slog.NewTextHandler(io.Discard, nil)),
		Retry:          DefaultRetry(),
	}, nil
}

func (c *Client) Clone() *Client {
	out := *c
	out.DefaultHeaders = c.DefaultHeaders.Clone()
	return &out
}

func (c *Client) Resolve(path string) string {
	// url.JoinPath would clean too aggressively for some base URLs with paths.
	u := *c.BaseURL
	u.Path = joinPath(u.Path, path)
	return u.String()
}

func joinPath(a, b string) string {
	if a == "" {
		return b
	}
	if b == "" {
		return a
	}
	if a[len(a)-1] == '/' {
		if b[0] == '/' {
			return a + b[1:]
		}
		return a + b
	}
	if b[0] == '/' {
		return a + b
	}
	return a + "/" + b
}

func (c *Client) DoJSON(ctx context.Context, method, path string, hdr http.Header, reqBody any) (*http.Response, []byte, error) {
	var bodyBytes []byte
	if reqBody != nil {
		b, err := json.Marshal(reqBody)
		if err != nil {
			return nil, nil, err
		}
		bodyBytes = b
	}

	attempts := c.Retry.MaxAttempts
	if attempts <= 0 {
		attempts = 1
	}

	for attempt := 1; attempt <= attempts; attempt++ {
		resp, raw, err := c.doOnce(ctx, method, path, hdr, bodyBytes)
		if err == nil {
			return resp, raw, nil
		}
		if attempt == attempts || !shouldRetry(err) {
			return nil, raw, err
		}

		sleep := backoff(c.Retry.InitialBackoff, c.Retry.MaxBackoff, attempt-1)
		c.Logger.Debug("llm http retry", "attempt", attempt, "sleep", sleep, "err", err)
		select {
		case <-ctx.Done():
			return nil, raw, ctx.Err()
		case <-time.After(sleep):
		}
	}

	return nil, nil, errors.New("unreachable")
}

func (c *Client) DoStream(ctx context.Context, method, path string, hdr http.Header, body any) (*http.Response, error) {
	var bodyBytes []byte
	if body != nil {
		b, err := json.Marshal(body)
		if err != nil {
			return nil, err
		}
		bodyBytes = b
	}
	return c.doStreamOnce(ctx, method, path, hdr, bodyBytes)
}

func (c *Client) doOnce(ctx context.Context, method, path string, hdr http.Header, bodyBytes []byte) (*http.Response, []byte, error) {
	urlStr := c.Resolve(path)
	req, err := http.NewRequestWithContext(ctx, method, urlStr, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, nil, err
	}

	mergeHeaders(req.Header, c.DefaultHeaders)
	mergeHeaders(req.Header, hdr)
	if c.UserAgent != "" && req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	if req.Header.Get("X-Request-Id") == "" {
		req.Header.Set("X-Request-Id", randomID())
	}
	if method == http.MethodPost && req.Header.Get("Idempotency-Key") == "" {
		req.Header.Set("Idempotency-Key", req.Header.Get("X-Request-Id"))
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, nil, err
	}
	defer resp.Body.Close()

	raw, err := io.ReadAll(resp.Body)
	if err != nil {
		return nil, nil, err
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp, raw, nil
	}
	return nil, raw, &HTTPStatusError{StatusCode: resp.StatusCode, Body: raw, Header: resp.Header.Clone()}
}

func (c *Client) doStreamOnce(ctx context.Context, method, path string, hdr http.Header, bodyBytes []byte) (*http.Response, error) {
	urlStr := c.Resolve(path)
	req, err := http.NewRequestWithContext(ctx, method, urlStr, bytes.NewReader(bodyBytes))
	if err != nil {
		return nil, err
	}

	mergeHeaders(req.Header, c.DefaultHeaders)
	mergeHeaders(req.Header, hdr)
	if c.UserAgent != "" && req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", c.UserAgent)
	}
	if req.Header.Get("X-Request-Id") == "" {
		req.Header.Set("X-Request-Id", randomID())
	}
	if method == http.MethodPost && req.Header.Get("Idempotency-Key") == "" {
		req.Header.Set("Idempotency-Key", req.Header.Get("X-Request-Id"))
	}

	resp, err := c.HTTPClient.Do(req)
	if err != nil {
		return nil, err
	}
	if resp.StatusCode >= 200 && resp.StatusCode < 300 {
		return resp, nil
	}
	defer resp.Body.Close()
	raw, _ := io.ReadAll(resp.Body)
	return nil, &HTTPStatusError{StatusCode: resp.StatusCode, Body: raw, Header: resp.Header.Clone()}
}

type HTTPStatusError struct {
	StatusCode int
	Body       []byte
	Header     http.Header
}

func (e *HTTPStatusError) Error() string {
	return http.StatusText(e.StatusCode)
}

func shouldRetry(err error) bool {
	if errors.Is(err, context.Canceled) || errors.Is(err, context.DeadlineExceeded) {
		return false
	}
	var se *HTTPStatusError
	if errors.As(err, &se) {
		switch se.StatusCode {
		case http.StatusTooManyRequests, http.StatusRequestTimeout:
			return true
		case http.StatusInternalServerError, http.StatusBadGateway, http.StatusServiceUnavailable, http.StatusGatewayTimeout:
			return true
		default:
			return false
		}
	}
	// network / io errors are generally retryable
	return true
}

func backoff(initial, max time.Duration, attempt int) time.Duration {
	if initial <= 0 {
		initial = 250 * time.Millisecond
	}
	if max <= 0 {
		max = 2 * time.Second
	}

	f := float64(initial) * math.Pow(2, float64(attempt))
	d := time.Duration(f)
	if d > max {
		d = max
	}

	// Add a small jitter.
	j := jitter(0.2)
	return time.Duration(float64(d) * (1 + j))
}

func jitter(maxFrac float64) float64 {
	if maxFrac <= 0 {
		return 0
	}
	n, err := rand.Int(rand.Reader, big.NewInt(1000))
	if err != nil {
		return 0
	}
	return (float64(n.Int64())/1000.0)*maxFrac - maxFrac/2
}

func mergeHeaders(dst, src http.Header) {
	for k, vs := range src {
		for _, v := range vs {
			dst.Add(k, v)
		}
	}
}

func randomID() string {
	var b [16]byte
	_, err := rand.Read(b[:])
	if err != nil {
		return ""
	}
	return hex.EncodeToString(b[:])
}
