package httpx

import (
	"bytes"
	"context"
	"encoding/json"
	"errors"
	"io"
	"net/http"
	"net/url"
	"strings"
	"time"
)

type RequestOption interface{ apply(*requestConfig) }

type requestOptionFunc func(*requestConfig)

func (f requestOptionFunc) apply(c *requestConfig) { f(c) }

type requestConfig struct {
	header http.Header
	query  url.Values

	timeout time.Duration

	body        io.Reader
	bodyBytes   []byte
	contentType string

	bearerToken string
	basicUser   string
	basicPass   string
}

func WithHeader(key, value string) RequestOption {
	return requestOptionFunc(func(c *requestConfig) {
		if c.header == nil {
			c.header = make(http.Header)
		}
		c.header.Set(key, value)
	})
}

func WithHeaders(h http.Header) RequestOption {
	return requestOptionFunc(func(c *requestConfig) {
		if h == nil {
			return
		}
		if c.header == nil {
			c.header = make(http.Header)
		}
		for k, vv := range h {
			for _, v := range vv {
				c.header.Add(k, v)
			}
		}
	})
}

func WithQuery(values url.Values) RequestOption {
	return requestOptionFunc(func(c *requestConfig) {
		if values == nil {
			return
		}
		if c.query == nil {
			c.query = make(url.Values)
		}
		for k, vv := range values {
			for _, v := range vv {
				c.query.Add(k, v)
			}
		}
	})
}

func WithQueryParam(key, value string) RequestOption {
	return requestOptionFunc(func(c *requestConfig) {
		if c.query == nil {
			c.query = make(url.Values)
		}
		c.query.Add(key, value)
	})
}

// WithRequestTimeout sets a per-request deadline upper bound.
// If the request context already has a deadline, the earlier one wins.
func WithRequestTimeout(d time.Duration) RequestOption {
	return requestOptionFunc(func(c *requestConfig) { c.timeout = d })
}

// WithBodyBytes sets the request body as bytes (retry-safe).
func WithBodyBytes(b []byte) RequestOption {
	return requestOptionFunc(func(c *requestConfig) {
		c.bodyBytes = append([]byte(nil), b...)
		c.body = nil
	})
}

// WithBody sets the request body reader. Note: this is not retry-safe unless req.GetBody is set.
func WithBody(r io.Reader) RequestOption {
	return requestOptionFunc(func(c *requestConfig) {
		c.body = r
		c.bodyBytes = nil
	})
}

// WithJSON sets the request body to a JSON-encoded value (retry-safe).
func WithJSON(v any) RequestOption {
	return requestOptionFunc(func(c *requestConfig) {
		b, err := json.Marshal(v)
		if err != nil {
			// Capture the error later during request build.
			c.body = errReader{err: err}
			c.bodyBytes = nil
			return
		}
		c.bodyBytes = b
		c.body = nil
		c.contentType = "application/json"
	})
}

func WithBearerToken(token string) RequestOption {
	return requestOptionFunc(func(c *requestConfig) { c.bearerToken = token })
}

func WithBasicAuth(user, pass string) RequestOption {
	return requestOptionFunc(func(c *requestConfig) {
		c.basicUser = user
		c.basicPass = pass
	})
}

type errReader struct{ err error }

func (r errReader) Read([]byte) (int, error) { return 0, r.err }

func (r errReader) Close() error { return nil }

type requestTimeoutKey struct{}

func withRequestTimeout(ctx context.Context, d time.Duration) context.Context {
	return context.WithValue(ctx, requestTimeoutKey{}, d)
}

func requestTimeout(ctx context.Context) time.Duration {
	if ctx == nil {
		return 0
	}
	v := ctx.Value(requestTimeoutKey{})
	if v == nil {
		return 0
	}
	if d, ok := v.(time.Duration); ok {
		return d
	}
	return 0
}

func (c *Client) NewRequest(ctx context.Context, method, path string, opts ...RequestOption) (*http.Request, error) {
	if ctx == nil {
		ctx = context.Background()
	}
	rc := requestConfig{}
	for _, o := range opts {
		if o != nil {
			o.apply(&rc)
		}
	}

	u, err := c.resolveURL(path, rc.query)
	if err != nil {
		return nil, err
	}

	if rc.timeout > 0 {
		ctx = withRequestTimeout(ctx, rc.timeout)
	}

	var body io.Reader
	if len(rc.bodyBytes) > 0 || rc.bodyBytes != nil {
		body = bytes.NewReader(rc.bodyBytes)
	} else if rc.body != nil {
		body = rc.body
	}

	req, err := http.NewRequestWithContext(ctx, strings.ToUpper(method), u.String(), body)
	if err != nil {
		return nil, err
	}
	if len(rc.bodyBytes) > 0 || rc.bodyBytes != nil {
		// Ensure retries can replay the body.
		b := append([]byte(nil), rc.bodyBytes...)
		req.GetBody = func() (io.ReadCloser, error) {
			return io.NopCloser(bytes.NewReader(b)), nil
		}
	}

	// Apply headers: default headers first, then request headers override.
	for k, vv := range c.defaultHeaders {
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}
	for k, vv := range rc.header {
		req.Header.Del(k)
		for _, v := range vv {
			req.Header.Add(k, v)
		}
	}
	if rc.contentType != "" && req.Header.Get("Content-Type") == "" {
		req.Header.Set("Content-Type", rc.contentType)
	}
	if c.userAgent != "" && req.Header.Get("User-Agent") == "" {
		req.Header.Set("User-Agent", c.userAgent)
	}
	if rc.bearerToken != "" && req.Header.Get("Authorization") == "" {
		req.Header.Set("Authorization", "Bearer "+rc.bearerToken)
	}
	if rc.basicUser != "" && req.Header.Get("Authorization") == "" {
		req.SetBasicAuth(rc.basicUser, rc.basicPass)
	}
	if c.requestID.Header != "" && req.Header.Get(c.requestID.Header) == "" {
		if c.requestID.New != nil {
			if id := strings.TrimSpace(c.requestID.New()); id != "" {
				req.Header.Set(c.requestID.Header, id)
			}
		}
	}

	// Surface JSON marshal errors (captured as body reader).
	if er, ok := rc.body.(errReader); ok && er.err != nil {
		return nil, er.err
	}
	if errors.Is(ctx.Err(), context.DeadlineExceeded) || errors.Is(ctx.Err(), context.Canceled) {
		return nil, ctx.Err()
	}
	return req, nil
}
