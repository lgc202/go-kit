package httpx

import (
	"bytes"
	"context"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type Client struct {
	httpClient *http.Client

	baseURL *url.URL

	timeout        time.Duration
	defaultHeaders http.Header
	userAgent      string

	retry      RetryConfig
	maxErrBody int64

	requestID RequestIDConfig

	rateLimiter RateLimiter
	before      []BeforeHook
	after       []AfterHook
}

// New constructs a Client from DefaultConfig() plus the provided options.
func New(opts ...Option) (*Client, error) {
	cfg := DefaultConfig()
	for _, o := range opts {
		if o != nil {
			o.apply(&cfg)
		}
	}
	return NewWithConfig(cfg)
}

func NewWithConfig(cfg Config) (*Client, error) {
	var bu *url.URL
	if strings.TrimSpace(cfg.BaseURL) != "" {
		u, err := url.Parse(strings.TrimSpace(cfg.BaseURL))
		if err != nil {
			return nil, err
		}
		if u.Scheme == "" || u.Host == "" {
			return nil, &url.Error{Op: "parse", URL: cfg.BaseURL, Err: errors.New("base url must be absolute")}
		}
		// Normalize so relative paths resolve as expected (treat BaseURL path as a prefix).
		if u.Path != "" && !strings.HasSuffix(u.Path, "/") {
			u.Path += "/"
		}
		bu = u
	}

	rt := cfg.Transport
	if rt == nil {
		rt = DefaultTransport()
	}

	hc := &http.Client{
		Transport: rt,
	}

	maxErrBody := cfg.MaxErrorBodyBytes
	if maxErrBody == 0 {
		maxErrBody = DefaultMaxErrorBodyBytes
	}

	// Clone headers to avoid caller mutation.
	hdr := make(http.Header)
	for k, vv := range cfg.DefaultHeaders {
		for _, v := range vv {
			hdr.Add(k, v)
		}
	}

	c := &Client{
		httpClient:     hc,
		baseURL:        bu,
		timeout:        cfg.Timeout,
		defaultHeaders: hdr,
		userAgent:      cfg.UserAgent,
		retry:          cfg.Retry,
		maxErrBody:     maxErrBody,
		requestID:      cfg.RequestID,
	}
	if c.requestID.New == nil && c.requestID.Header != "" {
		c.requestID.New = DefaultRequestID
	}
	if c.retry.Backoff == nil {
		c.retry.Backoff = DefaultBackoff()
	}
	return c, nil
}

// WithMiddleware wraps the underlying RoundTripper with middleware.
// Call this during initialization (before the client is used concurrently).
func (c *Client) WithMiddleware(mws ...Middleware) *Client {
	if len(mws) == 0 {
		return c
	}
	rt := c.httpClient.Transport
	if rt == nil {
		rt = http.DefaultTransport
	}
	c.httpClient.Transport = chain(rt, mws)
	return c
}

// WithRateLimiter installs a client-wide rate limiter.
func (c *Client) WithRateLimiter(rl RateLimiter) *Client {
	c.rateLimiter = rl
	return c
}

// WithHooks adds hooks (executed for every attempt).
func (c *Client) WithHooks(before []BeforeHook, after []AfterHook) *Client {
	c.before = append(c.before, before...)
	c.after = append(c.after, after...)
	return c
}

func (c *Client) resolveURL(path string, q url.Values) (*url.URL, error) {
	p := strings.TrimSpace(path)
	if p == "" {
		return nil, errors.New("empty url/path")
	}
	u, err := url.Parse(p)
	if err != nil {
		return nil, err
	}
	if u.IsAbs() {
		u2 := *u
		if q != nil {
			qq := u2.Query()
			for k, vv := range q {
				for _, v := range vv {
					qq.Add(k, v)
				}
			}
			u2.RawQuery = qq.Encode()
		}
		return &u2, nil
	}
	if c.baseURL == nil {
		return nil, errors.New("relative path requires BaseURL")
	}
	// Treat leading "/" as a relative path when BaseURL is set, so BaseURL with a path
	// prefix (e.g. https://host/api/v1) works with "/users" as expected.
	if strings.HasPrefix(u.Path, "/") {
		u2 := *u
		u2.Path = strings.TrimPrefix(u2.Path, "/")
		u = &u2
	}
	u2 := c.baseURL.ResolveReference(u)
	if q != nil {
		qq := u2.Query()
		for k, vv := range q {
			for _, v := range vv {
				qq.Add(k, v)
			}
		}
		u2.RawQuery = qq.Encode()
	}
	return u2, nil
}

func withEarlierDeadline(ctx context.Context, deadline time.Time) (context.Context, context.CancelFunc) {
	if deadline.IsZero() {
		return ctx, func() {}
	}
	if existing, ok := ctx.Deadline(); ok && !existing.After(deadline) {
		return ctx, func() {}
	}
	return context.WithDeadline(ctx, deadline)
}

func earliestDeadline(base context.Context, timeouts ...time.Duration) (time.Time, bool) {
	now := time.Now()
	var earliest time.Time
	for _, d := range timeouts {
		if d <= 0 {
			continue
		}
		dd := now.Add(d)
		if earliest.IsZero() || dd.Before(earliest) {
			earliest = dd
		}
	}
	if dl, ok := base.Deadline(); ok {
		if earliest.IsZero() || dl.Before(earliest) {
			earliest = dl
		}
	}
	if earliest.IsZero() {
		return time.Time{}, false
	}
	return earliest, true
}

// Do executes the request with retries (if configured). It mirrors net/http semantics:
// - transport errors are returned as error
// - non-2xx responses are returned as resp with nil error
func (c *Client) Do(req *http.Request) (*http.Response, error) {
	return c.do(req, false)
}

// DoStatus executes the request with retries and converts non-2xx responses into *Error.
// It reads up to MaxErrorBodyBytes from the response body and then closes it.
func (c *Client) DoStatus(req *http.Request) (*http.Response, error) {
	return c.do(req, true)
}

func (c *Client) do(req *http.Request, statusAsError bool) (*http.Response, error) {
	if req == nil {
		return nil, errors.New("nil request")
	}
	ctx := req.Context()
	if ctx == nil {
		ctx = context.Background()
	}
	if dl, ok := earliestDeadline(ctx, c.timeout, requestTimeout(ctx)); ok {
		ctx2, cancel := withEarlierDeadline(ctx, dl)
		defer cancel()
		ctx = ctx2
	}
	req = req.Clone(ctx)

	maxAttempts := c.retry.MaxAttempts
	if maxAttempts <= 0 {
		maxAttempts = 1
	}
	startAll := time.Now()

	var lastResp *http.Response
	var lastErr error

	for attempt := 1; attempt <= maxAttempts; attempt++ {
		if ctx.Err() != nil {
			return nil, ctx.Err()
		}
		if c.retry.MaxElapsed > 0 && time.Since(startAll) > c.retry.MaxElapsed {
			if !statusAsError {
				// Mirror net/http: return the last response (if any) alongside the last error (if any).
				if lastResp != nil || lastErr != nil {
					return lastResp, lastErr
				}
				return nil, context.DeadlineExceeded
			}

			// DoStatus must not turn "time budget exceeded" into a nil error.
			if lastResp != nil && lastErr == nil {
				retryable := c.retry.canRetryMethod(req.Method) && c.retry.canRetryStatus(lastResp.StatusCode)
				return responseToError(req, lastResp, c.requestID.Header, c.maxErrBody, retryable)
			}
			if lastResp != nil && lastResp.Body != nil {
				_ = lastResp.Body.Close()
			}
			cause := lastErr
			if cause == nil {
				cause = context.DeadlineExceeded
			}
			return nil, &Error{
				Method:     req.Method,
				URL:        req.URL.String(),
				StatusCode: 0,
				RequestID:  strings.TrimSpace(req.Header.Get(c.requestID.Header)),
				Cause:      cause,
				Retryable:  false,
			}
		}

		if attempt > 1 {
			// Only requests with a body need GetBody for retries.
			if req.Body != nil && req.Body != http.NoBody {
				if req.GetBody == nil {
					return nil, errors.New("httpx: request body is not replayable (missing req.GetBody)")
				}
				b, err := req.GetBody()
				if err != nil {
					return nil, err
				}
				req.Body = b
			}
		}

		if c.rateLimiter != nil {
			if err := c.rateLimiter.Wait(ctx); err != nil {
				return nil, err
			}
		}
		for _, h := range c.before {
			if h == nil {
				continue
			}
			if err := h(req, attempt); err != nil {
				return nil, err
			}
		}

		t0 := time.Now()
		resp, err := c.httpClient.Do(req)
		dur := time.Since(t0)

		for _, h := range c.after {
			if h != nil {
				h(req, resp, err, dur, attempt)
			}
		}

		// Success or non-retryable result.
		if err == nil && resp != nil {
			// Always treat < 400 as success.
			if resp.StatusCode < 400 {
				return resp, nil
			}
			// In "raw" mode, non-2xx responses are still returned, but we may retry
			// based on status code policy (e.g. 5xx/429).
			if !statusAsError && !c.retry.canRetryStatus(resp.StatusCode) {
				return resp, nil
			}
		}

		lastResp = resp
		lastErr = err

		// Decide whether to retry.
		retry := attempt < maxAttempts && c.retry.canRetryMethod(req.Method)
		if retry {
			if err != nil {
				retry = shouldRetryNetErr(err)
			} else if resp != nil {
				retry = c.retry.canRetryStatus(resp.StatusCode)
			} else {
				retry = false
			}
		}

		// If the request has a body, retries require req.GetBody to replay it.
		if retry && req.Body != nil && req.Body != http.NoBody && req.GetBody == nil {
			retry = false
		}

		if !retry {
			break
		}

		// Drain body for connection reuse before retrying.
		if resp != nil && resp.Body != nil {
			_, _ = io.Copy(io.Discard, io.LimitReader(resp.Body, 1<<20))
			_ = resp.Body.Close()
		}

		// Backoff.
		wait := c.retry.Backoff.Next(attempt)
		if c.retry.RespectRetryAfter && resp != nil && (resp.StatusCode == http.StatusTooManyRequests || resp.StatusCode == http.StatusServiceUnavailable) {
			if ra, ok := parseRetryAfter(resp, time.Now()); ok {
				wait = ra
				if c.retry.MaxRetryAfter > 0 && wait > c.retry.MaxRetryAfter {
					wait = c.retry.MaxRetryAfter
				}
			}
		}
		if err := sleep(ctx, wait); err != nil {
			return nil, err
		}
	}

	if !statusAsError {
		return lastResp, lastErr
	}
	if lastErr != nil {
		// http.Client may return a non-nil resp alongside an error (e.g. redirect issues).
		// Since DoStatus does not return resp on transport errors, ensure we don't leak bodies.
		if lastResp != nil && lastResp.Body != nil {
			_ = lastResp.Body.Close()
		}
		return nil, &Error{
			Method:     req.Method,
			URL:        req.URL.String(),
			StatusCode: 0,
			RequestID:  strings.TrimSpace(req.Header.Get(c.requestID.Header)),
			Cause:      lastErr,
			Retryable:  c.retry.canRetryMethod(req.Method) && shouldRetryNetErr(lastErr),
		}
	}
	if lastResp != nil {
		retryable := c.retry.canRetryMethod(req.Method) && c.retry.canRetryStatus(lastResp.StatusCode)
		return responseToError(req, lastResp, c.requestID.Header, c.maxErrBody, retryable)
	}
	return nil, errors.New("request failed")
}

func responseToError(req *http.Request, resp *http.Response, requestIDHeader string, maxErrBody int64, retryable bool) (*http.Response, error) {
	if resp == nil {
		return nil, &Error{Method: req.Method, URL: req.URL.String(), Cause: errors.New("nil response")}
	}
	if resp.Body != nil {
		defer resp.Body.Close()
	}

	var raw []byte
	if resp.Body != nil && maxErrBody != 0 {
		b, _ := io.ReadAll(io.LimitReader(resp.Body, maxErrBody))
		raw = b
	}

	// Expose the captured bytes to the caller (debuggability) but avoid holding open sockets.
	resp.Body = io.NopCloser(bytes.NewReader(raw))

	rid := ""
	if requestIDHeader != "" {
		rid = strings.TrimSpace(resp.Header.Get(requestIDHeader))
		if rid == "" {
			rid = strings.TrimSpace(req.Header.Get(requestIDHeader))
		}
	}
	ra, _ := parseRetryAfter(resp, time.Now())

	return resp, &Error{
		Method:     req.Method,
		URL:        req.URL.String(),
		StatusCode: resp.StatusCode,
		RequestID:  rid,
		RetryAfter: ra,
		RawBody:    raw,
		Retryable:  retryable,
		Cause:      errors.New(http.StatusText(resp.StatusCode)),
	}
}
