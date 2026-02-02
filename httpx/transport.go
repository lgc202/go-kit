package httpx

import (
	"net"
	"net/http"
	"net/url"
	"time"
)

// TransportConfig captures a subset of http.Transport knobs that commonly matter in production.
type TransportConfig struct {
	Proxy                 func(*http.Request) (*url.URL, error)
	DialTimeout           time.Duration
	DialKeepAlive         time.Duration
	TLSHandshakeTimeout   time.Duration
	ResponseHeaderTimeout time.Duration
	ExpectContinueTimeout time.Duration
	IdleConnTimeout       time.Duration

	MaxIdleConns        int
	MaxIdleConnsPerHost int
	MaxConnsPerHost     int
	ForceAttemptHTTP2   bool
}

// NewTransport builds an *http.Transport starting from DefaultTransport() and applying overrides.
func NewTransport(cfg TransportConfig) *http.Transport {
	t := DefaultTransport()
	if cfg.Proxy != nil {
		t.Proxy = cfg.Proxy
	}
	if cfg.DialTimeout > 0 || cfg.DialKeepAlive > 0 {
		d := &net.Dialer{
			Timeout:   cfg.DialTimeout,
			KeepAlive: cfg.DialKeepAlive,
		}
		t.DialContext = d.DialContext
	}
	if cfg.TLSHandshakeTimeout > 0 {
		t.TLSHandshakeTimeout = cfg.TLSHandshakeTimeout
	}
	if cfg.ResponseHeaderTimeout > 0 {
		t.ResponseHeaderTimeout = cfg.ResponseHeaderTimeout
	}
	if cfg.ExpectContinueTimeout > 0 {
		t.ExpectContinueTimeout = cfg.ExpectContinueTimeout
	}
	if cfg.IdleConnTimeout > 0 {
		t.IdleConnTimeout = cfg.IdleConnTimeout
	}
	if cfg.MaxIdleConns > 0 {
		t.MaxIdleConns = cfg.MaxIdleConns
	}
	if cfg.MaxIdleConnsPerHost > 0 {
		t.MaxIdleConnsPerHost = cfg.MaxIdleConnsPerHost
	}
	if cfg.MaxConnsPerHost > 0 {
		t.MaxConnsPerHost = cfg.MaxConnsPerHost
	}
	if cfg.ForceAttemptHTTP2 {
		t.ForceAttemptHTTP2 = true
	}
	return t
}

// DefaultTransport returns a tuned clone of http.DefaultTransport.
func DefaultTransport() *http.Transport {
	// http.DefaultTransport is a *http.Transport in stdlib.
	base, _ := http.DefaultTransport.(*http.Transport)
	if base == nil {
		return &http.Transport{}
	}
	t := base.Clone()

	// Safer defaults for services (not too aggressive, but avoids common footguns).
	t.DialContext = (&net.Dialer{
		Timeout:   5 * time.Second,
		KeepAlive: 30 * time.Second,
	}).DialContext
	t.TLSHandshakeTimeout = 5 * time.Second
	t.ResponseHeaderTimeout = 15 * time.Second
	t.ExpectContinueTimeout = 1 * time.Second
	t.IdleConnTimeout = 90 * time.Second
	if t.MaxIdleConns == 0 {
		t.MaxIdleConns = 200
	}
	if t.MaxIdleConnsPerHost == 0 {
		t.MaxIdleConnsPerHost = 50
	}
	t.ForceAttemptHTTP2 = true
	return t
}
