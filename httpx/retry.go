package httpx

import (
	"context"
	crand "crypto/rand"
	"encoding/binary"
	"errors"
	"math/rand/v2"
	"net"
	"net/http"
	"strconv"
	"strings"
	"sync"
	"time"
)

type RetryConfig struct {
	// MaxAttempts includes the initial attempt. If <= 1, retries are disabled.
	MaxAttempts int

	// MaxElapsed is the max total time spent across attempts (including backoff sleeps).
	// If zero, it is not enforced (but the request context/Client.Timeout still apply).
	MaxElapsed time.Duration

	// Methods lists HTTP methods eligible for retries.
	// If empty, a safe default of idempotent methods is used.
	Methods map[string]bool

	// StatusCodes lists response status codes eligible for retries.
	// If empty, a safe default set is used.
	StatusCodes map[int]bool

	// Backoff computes the sleep duration before the next retry.
	// If nil, DefaultBackoff() is used.
	Backoff Backoff

	// RespectRetryAfter uses Retry-After header as the backoff for 429/503 when present.
	RespectRetryAfter bool

	// MaxRetryAfter caps Retry-After. If zero, no cap is applied.
	MaxRetryAfter time.Duration
}

func DefaultRetryConfig() RetryConfig {
	return RetryConfig{
		MaxAttempts:       3,
		MaxElapsed:        0,
		Methods:           defaultRetryMethods(),
		StatusCodes:       defaultRetryStatusCodes(),
		Backoff:           DefaultBackoff(),
		RespectRetryAfter: true,
		MaxRetryAfter:     30 * time.Second,
	}
}

func defaultRetryMethods() map[string]bool {
	return map[string]bool{
		http.MethodGet:     true,
		http.MethodHead:    true,
		http.MethodPut:     true,
		http.MethodDelete:  true,
		http.MethodOptions: true,
		http.MethodTrace:   true,
	}
}

func defaultRetryStatusCodes() map[int]bool {
	return map[int]bool{
		http.StatusTooManyRequests:     true,
		http.StatusRequestTimeout:      true,
		http.StatusInternalServerError: true,
		http.StatusBadGateway:          true,
		http.StatusServiceUnavailable:  true,
		http.StatusGatewayTimeout:      true,
	}
}

type Backoff interface {
	// Next returns how long to sleep before retrying attempt+1.
	// attempt starts at 1 for the first retry (i.e. after the first failed request).
	Next(attempt int) time.Duration
}

type ExponentialBackoff struct {
	Base   time.Duration
	Max    time.Duration
	Jitter float64 // 0..1
}

var (
	jitterMu  sync.Mutex
	jitterRng = rand.New(rand.NewPCG(seed64(), seed64()))
)

func seed64() uint64 {
	var b [8]byte
	if _, err := crand.Read(b[:]); err == nil {
		return binary.LittleEndian.Uint64(b[:])
	}
	// Fallback: time-based seed (still better than deterministic).
	return uint64(time.Now().UnixNano())
}

func jitterFloat64() float64 {
	jitterMu.Lock()
	defer jitterMu.Unlock()
	return jitterRng.Float64()
}

func DefaultBackoff() Backoff {
	return ExponentialBackoff{
		Base:   200 * time.Millisecond,
		Max:    3 * time.Second,
		Jitter: 0.2,
	}
}

func (b ExponentialBackoff) Next(attempt int) time.Duration {
	if attempt < 1 {
		attempt = 1
	}
	base := b.Base
	if base <= 0 {
		base = 200 * time.Millisecond
	}
	max := b.Max
	if max <= 0 {
		max = 3 * time.Second
	}

	// base * 2^(attempt-1)
	d := base
	for i := 1; i < attempt; i++ {
		if d >= max/2 {
			d = max
			break
		}
		d *= 2
	}
	if d > max {
		d = max
	}

	j := b.Jitter
	if j <= 0 {
		return d
	}
	if j > 1 {
		j = 1
	}

	// +/- jitter%
	f := 1 + (jitterFloat64()*2-1)*j
	if f < 0 {
		f = 0
	}
	return time.Duration(float64(d) * f)
}

func (c RetryConfig) canRetryMethod(method string) bool {
	if c.MaxAttempts <= 1 {
		return false
	}
	m := strings.ToUpper(strings.TrimSpace(method))
	if m == "" {
		return false
	}
	methods := c.Methods
	if len(methods) == 0 {
		methods = defaultRetryMethods()
	}
	return methods[m]
}

func (c RetryConfig) canRetryStatus(code int) bool {
	statuses := c.StatusCodes
	if len(statuses) == 0 {
		statuses = defaultRetryStatusCodes()
	}
	return statuses[code]
}

func shouldRetryNetErr(err error) bool {
	var ne net.Error
	if errors.As(err, &ne) {
		return ne.Timeout() || ne.Temporary()
	}
	return false
}

func parseRetryAfter(resp *http.Response, now time.Time) (time.Duration, bool) {
	if resp == nil {
		return 0, false
	}
	v := strings.TrimSpace(resp.Header.Get("Retry-After"))
	if v == "" {
		return 0, false
	}
	if secs, err := strconv.Atoi(v); err == nil && secs >= 0 {
		return time.Duration(secs) * time.Second, true
	}
	if t, err := http.ParseTime(v); err == nil {
		d := t.Sub(now)
		if d < 0 {
			d = 0
		}
		return d, true
	}
	return 0, false
}

func sleep(ctx context.Context, d time.Duration) error {
	if d <= 0 {
		return nil
	}
	t := time.NewTimer(d)
	defer t.Stop()
	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-t.C:
		return nil
	}
}
