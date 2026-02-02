package httpx

import (
	"context"
	"net/http"
	"time"
)

// RateLimiter can be used to throttle outgoing requests.
// It should block until a token is available or ctx is canceled.
type RateLimiter interface {
	Wait(ctx context.Context) error
}

type BeforeHook func(req *http.Request, attempt int) error

type AfterHook func(req *http.Request, resp *http.Response, err error, dur time.Duration, attempt int)

type Middleware func(next http.RoundTripper) http.RoundTripper

func chain(rt http.RoundTripper, mws []Middleware) http.RoundTripper {
	for i := len(mws) - 1; i >= 0; i-- {
		if mws[i] == nil {
			continue
		}
		rt = mws[i](rt)
	}
	return rt
}
