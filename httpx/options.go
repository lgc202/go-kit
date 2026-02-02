package httpx

import (
	"net/http"
	"time"
)

type Option interface{ apply(*Config) }

type optionFunc func(*Config)

func (f optionFunc) apply(c *Config) { f(c) }

func WithBaseURL(baseURL string) Option {
	return optionFunc(func(c *Config) { c.BaseURL = baseURL })
}

func WithTimeout(d time.Duration) Option {
	return optionFunc(func(c *Config) { c.Timeout = d })
}

func WithTransport(rt http.RoundTripper) Option {
	return optionFunc(func(c *Config) { c.Transport = rt })
}

func WithDefaultHeader(key, value string) Option {
	return optionFunc(func(c *Config) {
		if c.DefaultHeaders == nil {
			c.DefaultHeaders = make(http.Header)
		}
		c.DefaultHeaders.Set(key, value)
	})
}

func WithDefaultHeaders(h http.Header) Option {
	return optionFunc(func(c *Config) {
		if h == nil {
			return
		}
		if c.DefaultHeaders == nil {
			c.DefaultHeaders = make(http.Header)
		}
		for k, vv := range h {
			for _, v := range vv {
				c.DefaultHeaders.Add(k, v)
			}
		}
	})
}

func WithUserAgent(ua string) Option {
	return optionFunc(func(c *Config) { c.UserAgent = ua })
}

func WithRetry(cfg RetryConfig) Option {
	return optionFunc(func(c *Config) { c.Retry = cfg })
}

func WithMaxErrorBodyBytes(n int64) Option {
	return optionFunc(func(c *Config) { c.MaxErrorBodyBytes = n })
}

func WithRequestID(cfg RequestIDConfig) Option {
	return optionFunc(func(c *Config) { c.RequestID = cfg })
}
